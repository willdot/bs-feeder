package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/willdot/bskyfeedgen/frontend"
	"github.com/willdot/bskyfeedgen/store"
)

const (
	bskyBaseURL = "https://bsky.social/xrpc"
)

type loginRequest struct {
	Handle      string `json:"handle"`
	AppPassword string `json:"appPassword"`
}

type BskyAuth struct {
	AccessJwt string `json:"accessJwt"`
	Did       string `json:"did"`
}

func (s *Server) HandleSubscriptions(w http.ResponseWriter, r *http.Request) {
	didCookie, err := r.Cookie(didCookieName)
	if err != nil {
		slog.Error("read DID cookie", "error", err)
		frontend.Login("", "").Render(r.Context(), w)
		return
	}
	if didCookie == nil {
		slog.Error("missing DID cookie")
		frontend.Login("", "").Render(r.Context(), w)
		return
	}

	usersDid := didCookie.Value

	slog.Info("did request", "did", usersDid)

	subs, err := s.feeder.GetSubscriptionsForUser(r.Context(), usersDid)
	if err != nil {
		slog.Error("error getting subscriptions for user", "error", err)
		frontend.Subscriptions("failed to get subscriptions", nil).Render(r.Context(), w)
		return
	}

	subResp := make([]store.Subscription, 0, len(subs))
	for _, sub := range subs {
		splitStr := strings.Split(sub.SubscribedPostURI, "/")

		if len(splitStr) != 5 {
			slog.Error("subscription URI was not expected - expected to have 5 strings after spliting by /", "uri", sub.SubscribedPostURI)
			continue
		}

		did := splitStr[2]

		handle, err := resolveDid(did)
		if err != nil {
			slog.Error("resolving did", "error", err, "did", did)
			handle = did
		}

		slog.Info("sub id", "id", sub.ID)

		uri := fmt.Sprintf("https://bsky.app/profile/%s/post/%s", handle, splitStr[4])
		sub.SubscribedPostURI = uri
		subResp = append(subResp, sub)
	}

	frontend.Subscriptions("", subResp).Render(r.Context(), w)
}

func (s *Server) HandleDeleteSubscription(w http.ResponseWriter, r *http.Request) {
	sub := r.PathValue("id")

	slog.Info("deleting sub", "sub", sub)

	didCookie, err := r.Cookie(didCookieName)
	if err != nil {
		slog.Error("read DID cookie", "error", err)
		frontend.Login("", "").Render(r.Context(), w)
		return
	}
	if didCookie == nil {
		slog.Error("missing DID cookie")
		frontend.Login("", "").Render(r.Context(), w)
		return
	}

	usersDid := didCookie.Value

	id, err := strconv.Atoi(sub)
	if err != nil {
		slog.Error("failed to convert sub ID to int", "error", err)
		http.Error(w, "invalid ID", http.StatusBadRequest)
		return
	}

	err = s.feeder.DeleteSubscriptionByIdAndUser(usersDid, id)
	if err != nil {
		slog.Error("delete subscription for user", "error", err, "subscription URI", sub)
		http.Error(w, "failed to delete subscription", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte("{}"))
}

func (s *Server) HandleLogin(w http.ResponseWriter, r *http.Request) {
	b, err := io.ReadAll(r.Body)
	if err != nil {
		slog.Error("failed to read body", "error", err)
		frontend.LoginForm("", "bad request").Render(r.Context(), w)
		return
	}

	var loginReq loginRequest
	err = json.Unmarshal(b, &loginReq)
	if err != nil {
		slog.Error("failed to unmarshal body", "error", err)
		frontend.LoginForm("", "bad request").Render(r.Context(), w)
		return
	}
	url := fmt.Sprintf("%s/com.atproto.server.createsession", bskyBaseURL)

	requestData := map[string]interface{}{
		"identifier": loginReq.Handle,
		"password":   loginReq.AppPassword,
	}

	data, err := json.Marshal(requestData)
	if err != nil {
		slog.Error("failed marshal POST request to sign into Bsky", "error", err)
		frontend.LoginForm(loginReq.Handle, "internal error").Render(r.Context(), w)
		return
	}

	reader := bytes.NewReader(data)

	req, err := http.NewRequest("POST", url, reader)
	if err != nil {
		slog.Error("failed to create POST request to sign into Bsky", "error", err)
		frontend.LoginForm(loginReq.Handle, "internal error").Render(r.Context(), w)
		return
	}

	req.Header.Add("Content-Type", "application/json")

	// TODO: create a client somewhere
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		slog.Error("failed to make POST request to sign into Bsky", "error", err)
		frontend.LoginForm(loginReq.Handle, "internal error").Render(r.Context(), w)
		return
	}

	defer res.Body.Close()

	slog.Info("bsky resp", "code", res.StatusCode)

	if res.StatusCode != 200 {
		slog.Error("failed to log into bluesky", "status code", res.StatusCode)
		frontend.LoginForm(loginReq.Handle, "not authorized").Render(r.Context(), w)
		return
	}

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		slog.Error("failed read response from Bsky login", "error", err)
		frontend.LoginForm(loginReq.Handle, "internal error").Render(r.Context(), w)
		return
	}

	var loginResp BskyAuth
	err = json.Unmarshal(resBody, &loginResp)
	if err != nil {
		slog.Error("failed unmarshal response from Bsky login", "error", err)
		frontend.LoginForm(loginReq.Handle, "internal error").Render(r.Context(), w)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:  jwtCookieName,
		Value: loginResp.AccessJwt,
	})

	http.SetCookie(w, &http.Cookie{
		Name:  didCookieName,
		Value: loginResp.Did,
	})

	ctx := context.WithValue(r.Context(), frontend.ContextUsernameKey, loginReq.Handle)
	r = r.WithContext(ctx)

	http.Redirect(w, r, "/", http.StatusOK)
}

func resolveDid(did string) (string, error) {
	resp, err := http.DefaultClient.Get(fmt.Sprintf("https://plc.directory/%s", did))
	if err != nil {
		return "", fmt.Errorf("error making request to resolve did: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("got response %d", resp.StatusCode)
	}

	type resolvedDid struct {
		Aka []string `json:"alsoKnownAs"`
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading response body: %w", err)
	}

	var resolved resolvedDid
	err = json.Unmarshal(b, &resolved)
	if err != nil {
		return "", fmt.Errorf("decode response body: %w", err)
	}

	if len(resolved.Aka) == 0 {
		return "", nil
	}

	res := strings.ReplaceAll(resolved.Aka[0], "at://", "")

	return res, nil
}
