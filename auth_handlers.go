package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/willdot/bskyfeedgen/frontend"
)

const (
	bskyBaseURL   = "https://bsky.social/xrpc"
	jwtCookieName = "JWT"
	didCookieName = "DID"
)

type loginRequest struct {
	Handle      string `json:"handle"`
	AppPassword string `json:"appPassword"`
}

type BskyAuth struct {
	AccessJwt string `json:"accessJwt"`
	Did       string `json:"did"`
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
	url := fmt.Sprintf("%s/com.atproto.server.createsession", bskyBaseURL)

	requestData := map[string]interface{}{
		"identifier": loginReq.Handle,
		"password":   loginReq.AppPassword,
	}

	data, err := json.Marshal(requestData)
	if err != nil {
		slog.Error("failed marshal POST request to sign into Bsky", "error", err)
		_ = frontend.LoginForm(loginReq.Handle, "internal error").Render(r.Context(), w)
		return
	}

	reader := bytes.NewReader(data)

	req, err := http.NewRequest("POST", url, reader)
	if err != nil {
		slog.Error("failed to create POST request to sign into Bsky", "error", err)
		_ = frontend.LoginForm(loginReq.Handle, "internal error").Render(r.Context(), w)
		return
	}

	req.Header.Add("Content-Type", "application/json")

	// TODO: create a client somewhere
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		slog.Error("failed to make POST request to sign into Bsky", "error", err)
		_ = frontend.LoginForm(loginReq.Handle, "internal error").Render(r.Context(), w)
		return
	}

	defer res.Body.Close()

	slog.Info("bsky resp", "code", res.StatusCode)

	if res.StatusCode != 200 {
		slog.Error("failed to log into bluesky", "status code", res.StatusCode)
		_ = frontend.LoginForm(loginReq.Handle, "not authorized").Render(r.Context(), w)
		return
	}

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		slog.Error("failed read response from Bsky login", "error", err)
		_ = frontend.LoginForm(loginReq.Handle, "internal error").Render(r.Context(), w)
		return
	}

	var loginResp BskyAuth
	err = json.Unmarshal(resBody, &loginResp)
	if err != nil {
		slog.Error("failed unmarshal response from Bsky login", "error", err)
		_ = frontend.LoginForm(loginReq.Handle, "internal error").Render(r.Context(), w)
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

	http.Redirect(w, r, "/", http.StatusOK)
}

func (s *Server) HandleSignOut(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:  jwtCookieName,
		Value: "",
	})

	http.SetCookie(w, &http.Cookie{
		Name:  didCookieName,
		Value: "",
	})

	_ = frontend.Login("", "").Render(r.Context(), w)
}

func (s *Server) authMiddleware(next func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		jwtCookie, err := r.Cookie(jwtCookieName)
		if err != nil {
			slog.Error("read JWT cookie", "error", err)
			_ = frontend.Login("", "").Render(r.Context(), w)
			return
		}
		if jwtCookie == nil {
			slog.Error("missing JWT cookie")
			_ = frontend.Login("", "").Render(r.Context(), w)
			return
		}

		didCookie, err := r.Cookie(didCookieName)
		if err != nil {
			slog.Error("read DID cookie", "error", err)
			_ = frontend.Login("", "").Render(r.Context(), w)
			return
		}
		if didCookie == nil {
			slog.Error("missing DID cookie")
			_ = frontend.Login("", "").Render(r.Context(), w)
			return
		}

		claims := jwt.MapClaims{}
		_, _, err = jwt.NewParser().ParseUnverified(jwtCookie.Value, &claims)
		if err != nil {
			slog.Error("parsing JWT", "error", err)
			_ = frontend.Login("", "").Render(r.Context(), w)
			return
		}

		if expiry, ok := claims["exp"].(string); ok {
			expiryInt, err := strconv.Atoi(expiry)
			if err != nil {
				slog.Error("invalid claims from token", "error", err)
				_ = frontend.Login("", "").Render(r.Context(), w)
				return
			}

			if time.Now().Unix() > int64(expiryInt) {
				_ = frontend.Login("", "").Render(r.Context(), w)
				return
			}
		}
		next(w, r)
	}
}
