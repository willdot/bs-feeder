package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/gorilla/sessions"
	oauth "github.com/haileyok/atproto-oauth-golang"
	oauthhelpers "github.com/haileyok/atproto-oauth-golang/helpers"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/willdot/bskyfeedgen/frontend"
	"github.com/willdot/bskyfeedgen/store"
)

const (
	scope = "atproto transition:generic"
)

func (s *Server) serverJwks(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(s.jwks.public)
}

func serveClientMetadata(w http.ResponseWriter, r *http.Request) {
	metadata := map[string]any{
		"client_id":                       fmt.Sprintf("%s/client-metadata.json", serverBase),
		"client_name":                     "BS Feeder",
		"client_uri":                      serverBase,
		"redirect_uris":                   []string{fmt.Sprintf("%s/oauth-callback", serverBase)},
		"grant_types":                     []string{"authorization_code", "refresh_token"},
		"response_types":                  []string{"code"},
		"application_type":                "web",
		"dpop_bound_access_tokens":        true,
		"jwks_uri":                        fmt.Sprintf("%s/jwks.json", serverBase),
		"scope":                           "atproto transition:generic",
		"token_endpoint_auth_method":      "private_key_jwt",
		"token_endpoint_auth_signing_alg": "ES256",
	}

	b, err := json.Marshal(metadata)
	if err != nil {
		slog.Error("failed to marshal client metadata", "error", err)
		http.Error(w, "marshal response", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(b)
}

func (s *Server) HandleLogin(w http.ResponseWriter, r *http.Request) {
	b, err := io.ReadAll(r.Body)
	if err != nil {
		slog.Error("failed to read body", "error", err)
		_ = frontend.Login("", "bad request").Render(r.Context(), w)
		return
	}

	var loginReq loginRequest
	err = json.Unmarshal(b, &loginReq)
	if err != nil {
		slog.Error("failed to unmarshal body", "error", err)
		_ = frontend.Login("", "bad request").Render(r.Context(), w)
		return
	}

	usersDID, err := resolveHandle(loginReq.Handle)
	if err != nil {
		slog.Error("resolve users handle", "error", err)
		_ = frontend.Login("", "bad request").Render(r.Context(), w)
		return
	}

	dpopPrivateKey, err := oauthhelpers.GenerateKey(nil)
	if err != nil {
		slog.Error("generate key", "error", err)
		_ = frontend.Login("", "internal server errror").Render(r.Context(), w)
		return
	}

	parResp, meta, err := s.parseLoginRequest(r.Context(), usersDID, loginReq.Handle, dpopPrivateKey)
	if err != nil {
		slog.Error("handle login request", "error", err)
		_ = frontend.Login("", "internal server errror").Render(r.Context(), w)
		return
	}

	dpopPrivateKeyJson, err := json.Marshal(dpopPrivateKey)
	if err != nil {
		slog.Error("marshal key", "error", err)
		_ = frontend.Login("", "internal server errror").Render(r.Context(), w)
		return
	}

	oauthRequst := store.OauthRequest{
		AuthserverIss:       meta.Issuer,
		State:               parResp.State,
		Did:                 usersDID,
		PkceVerifier:        parResp.PkceVerifier,
		DpopAuthserverNonce: parResp.DpopAuthserverNonce,
		DpopPrivateJwk:      string(dpopPrivateKeyJson),
	}
	err = s.oauthRequestStore.CreateOauthRequest(oauthRequst)
	if err != nil {
		// TODO: catch already exists
		slog.Error("create oauth request in store", "error", err)
		_ = frontend.Login("", "internal server errror").Render(r.Context(), w)
		return
	}

	u, _ := url.Parse(meta.AuthorizationEndpoint)
	u.RawQuery = fmt.Sprintf("client_id=%s&request_uri=%s", url.QueryEscape(fmt.Sprintf("%s/client-metadata.json", serverBase)), parResp.RequestUri)

	// ignore error here as it only returns an error for decoding an existing session but it will always return a session anyway which
	// is what we want
	session, _ := s.sessionStore.Get(r, "oauth-session")
	session.Values = map[interface{}]interface{}{}

	session.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   300, // save for five minutes
		HttpOnly: true,
	}

	session.Values["oauth_state"] = parResp.State
	session.Values["oauth_did"] = usersDID

	err = session.Save(r, w)
	if err != nil {
		slog.Error("save session", "error", err)
		_ = frontend.Login("", "internal server errror").Render(r.Context(), w)
		return
	}

	w.Header().Add("HX-Redirect", "/")
	http.Redirect(w, r, u.String(), http.StatusOK)
}

func (s *Server) parseLoginRequest(ctx context.Context, did, handle string, dpopPrivateKey jwk.Key) (*oauth.SendParAuthResponse, *oauth.OauthAuthorizationMetadata, error) {
	service, err := resolveService(ctx, did)
	if err != nil {
		return nil, nil, err
	}

	authserver, err := s.oauthClient.ResolvePdsAuthServer(ctx, service)
	if err != nil {
		return nil, nil, err
	}

	meta, err := s.oauthClient.FetchAuthServerMetadata(ctx, authserver)
	if err != nil {
		return nil, nil, err
	}

	resp, err := s.oauthClient.SendParAuthRequest(ctx, authserver, meta, handle, scope, dpopPrivateKey)
	if err != nil {
		return nil, nil, err
	}
	return resp, meta, nil
}

func (s *Server) handleOauthCallback(w http.ResponseWriter, r *http.Request) {
	resState := r.FormValue("state")
	resIss := r.FormValue("iss")
	resCode := r.FormValue("code")

	session, err := s.sessionStore.Get(r, "oauth-session")
	if err != nil {
		slog.Error("getting session", "error", err)
		_ = frontend.Login("", "internal server error").Render(r.Context(), w)
		return
	}

	sessionState := session.Values["oauth_state"]

	if resState == "" || resIss == "" || resCode == "" {
		slog.Error("request missing needed parameters")
		_ = frontend.Login("", "internal server error").Render(r.Context(), w)
		return
	}

	if resState != sessionState {
		slog.Error("session state does not match response state")
		_ = frontend.Login("", "internal server error").Render(r.Context(), w)
		return
	}

	oauthRequest, err := s.oauthRequestStore.GetOauthRequest(fmt.Sprintf("%s", sessionState))
	if err != nil {
		slog.Error("get oauth request from store", "error", err)
		_ = frontend.Login("", "internal server errror").Render(r.Context(), w)
		return
	}

	err = s.oauthRequestStore.DeleteOauthRequest(fmt.Sprintf("%s", sessionState))
	if err != nil {
		slog.Error("delete oauth request from store", "error", err)
		_ = frontend.Login("", "internal server errror").Render(r.Context(), w)
		return
	}

	jwk, err := oauthhelpers.ParseJWKFromBytes([]byte(oauthRequest.DpopPrivateJwk))
	if err != nil {
		slog.Error("parse JWK", "error", err)
		_ = frontend.Login("", "internal server errror").Render(r.Context(), w)
		return
	}

	initialTokenResp, err := s.oauthClient.InitialTokenRequest(r.Context(), resCode, resIss, oauthRequest.PkceVerifier, oauthRequest.DpopAuthserverNonce, jwk)
	if err != nil {
		slog.Error("getting token from request", "error", err)
		_ = frontend.Login("", "internal server error").Render(r.Context(), w)
		return
	}

	if initialTokenResp.Scope != scope {
		slog.Error("did not receive correct scopes from token request")
		_ = frontend.Login("", "internal server errror").Render(r.Context(), w)
		return
	}

	session.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7,
		HttpOnly: true,
	}

	// make sure the session is empty
	session.Values = map[interface{}]interface{}{}
	session.Values["did"] = oauthRequest.Did

	err = session.Save(r, w)
	if err != nil {
		slog.Error("save session", "error", err)
		_ = frontend.Login("", "internal server errror").Render(r.Context(), w)
		return
	}

	w.Header().Add("HX-Redirect", "/bookmarks")
	http.Redirect(w, r, "/bookmarks", http.StatusOK)
}

func (s *Server) HandleSignOut(w http.ResponseWriter, r *http.Request) {
	session, err := s.sessionStore.Get(r, "oauth-session")
	if err != nil {
		slog.Error("getting session", "error", err)
		_ = frontend.Login("", "internal server error").Render(r.Context(), w)
		return
	}
	session.Values = map[interface{}]interface{}{}
	session.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	}

	err = session.Save(r, w)
	if err != nil {
		slog.Error("save session", "error", err)
		_ = frontend.Login("", "internal server errror").Render(r.Context(), w)
		return
	}

	_ = frontend.Login("", "").Render(r.Context(), w)
}

func resolveService(ctx context.Context, did string) (string, error) {
	type Identity struct {
		Service []struct {
			ID              string `json:"id"`
			Type            string `json:"type"`
			ServiceEndpoint string `json:"serviceEndpoint"`
		} `json:"service"`
	}

	var ustr string
	if strings.HasPrefix(did, "did:plc:") {
		ustr = fmt.Sprintf("https://plc.directory/%s", did)
	} else if strings.HasPrefix(did, "did:web:") {
		ustr = fmt.Sprintf("https://%s/.well-known/did.json", strings.TrimPrefix(did, "did:web:"))
	} else {
		return "", fmt.Errorf("did was not a supported did type")
	}

	req, err := http.NewRequestWithContext(ctx, "GET", ustr, nil)
	if err != nil {
		return "", err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		io.Copy(io.Discard, resp.Body)
		return "", fmt.Errorf("could not find identity in plc registry")
	}

	var identity Identity
	if err := json.NewDecoder(resp.Body).Decode(&identity); err != nil {
		return "", err
	}

	var service string
	for _, svc := range identity.Service {
		if svc.ID == "#atproto_pds" {
			service = svc.ServiceEndpoint
		}
	}

	if service == "" {
		return "", fmt.Errorf("could not find atproto_pds service in identity services")
	}

	return service, nil
}
