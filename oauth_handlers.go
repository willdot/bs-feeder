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

	oauth "github.com/haileyok/atproto-oauth-golang"
	oauthhelpers "github.com/haileyok/atproto-oauth-golang/helpers"
	"github.com/willdot/bskyfeedgen/frontend"
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
		"client_id":   fmt.Sprintf("%s/client-metadata.json", serverBase),
		"client_name": "BS Feeder",
		"client_uri":  serverBase,
		// "logo_uri":                        fmt.Sprintf("%s/logo.png", serverUrlRoot),
		// "tos_uri":                         fmt.Sprintf("%s/tos", serverUrlRoot),
		// "policy_url":                      fmt.Sprintf("%s/policy", serverUrlRoot),
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

func (s *Server) handleOauthCallback(w http.ResponseWriter, r *http.Request) {
	resState := r.FormValue("state")
	resIss := r.FormValue("iss")
	resCode := r.FormValue("code")

	slog.Info("callback", "res state", resState, "res iss", resIss, "res code", resCode)
}

func (s *Server) HandleLoginTemp(w http.ResponseWriter, r *http.Request) {
	loginReq := loginRequest{
		Handle: "willdot.net",
	}
	parResp, meta, err := s.parseLoginRequest(r.Context(), loginReq)
	if err != nil {
		slog.Error("handle login request", "error", err)
		_ = frontend.LoginForm("", "internal server errror").Render(r.Context(), w)
		return
	}

	u, _ := url.Parse(meta.AuthorizationEndpoint)
	u.RawQuery = fmt.Sprintf("client_id=%s&request_uri=%s", url.QueryEscape(fmt.Sprintf("%s/client-metadata.json", serverBase)), parResp.RequestUri)

	slog.Info("redirect to", "url", u.String())

	http.Redirect(w, r, u.String(), http.StatusFound)
}

func (s *Server) HandleLogin(w http.ResponseWriter, r *http.Request) {
	b, err := io.ReadAll(r.Body)
	if err != nil {
		slog.Error("failed to read body", "error", err)
		_ = frontend.LoginForm("", "bad request").Render(r.Context(), w)
		return
	}

	var loginReq loginRequest
	err = json.Unmarshal(b, &loginReq)
	if err != nil {
		slog.Error("failed to unmarshal body", "error", err)
		_ = frontend.LoginForm("", "bad request").Render(r.Context(), w)
		return
	}

	parResp, meta, err := s.parseLoginRequest(r.Context(), loginReq)
	if err != nil {
		slog.Error("handle login request", "error", err)
		_ = frontend.LoginForm("", "internal server errror").Render(r.Context(), w)
		return
	}

	u, _ := url.Parse(meta.AuthorizationEndpoint)
	u.RawQuery = fmt.Sprintf("client_id=%s&request_uri=%s", url.QueryEscape(fmt.Sprintf("%s/client-metadata.json", serverBase)), parResp.RequestUri)

	slog.Info("redirect to", "url", u.String())

	http.Redirect(w, r, u.String(), http.StatusFound)
}

func (s *Server) parseLoginRequest(ctx context.Context, req loginRequest) (*oauth.SendParAuthResponse, *oauth.OauthAuthorizationMetadata, error) {
	//TODO: get did from handle
	service, err := resolveService(ctx, "did:plc:dadhhalkfcq3gucaq25hjqon")
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

	dpopPrivateKey, err := oauthhelpers.GenerateKey(nil)
	if err != nil {
		return nil, nil, err
	}

	// dpopPrivateKeyJson, err := json.Marshal(dpopPrivateKey)
	// if err != nil {
	// 	return nil, err
	// }

	resp, err := s.oauthClient.SendParAuthRequest(ctx, authserver, meta, req.Handle, scope, dpopPrivateKey)
	if err != nil {
		return nil, nil, err
	}
	return resp, meta, nil
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
