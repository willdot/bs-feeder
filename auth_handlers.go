package main

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/willdot/bskyfeedgen/frontend"
)

const (
	bskyBaseURL = "https://bsky.social/xrpc"
)

type loginRequest struct {
	Handle string `json:"handle"`
}

func (s *Server) authMiddleware(next func(http.ResponseWriter, *http.Request, string)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		did, ok := s.getDidFromSession(r)
		if !ok {
			_ = frontend.Login("", "").Render(r.Context(), w)
			return
		}

		next(w, r, did)
	}
}

func (s *Server) getDidFromSession(r *http.Request) (string, bool) {
	session, err := s.sessionStore.Get(r, "oauth-session")
	if err != nil {
		slog.Error("getting session", "error", err)
		return "", false
	}

	did, ok := session.Values["did"]
	if !ok {
		return "", false
	}

	return fmt.Sprintf("%s", did), true
}
