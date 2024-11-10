package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/bluesky-social/indigo/atproto/identity"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/golang-jwt/jwt/v5"
)

type Feeder interface {
	GetFeed(ctx context.Context, userDID, feed, cursor string, limit int) (*FeedReponse, error)
}

type Server struct {
	httpsrv     *http.Server
	feeder      Feeder
	feedHost    string
	feedDidBase string
}

func NewServer(port int, feeder Feeder, feedHost, feedDidBase string) *Server {
	srv := &Server{
		feeder:      feeder,
		feedHost:    feedHost,
		feedDidBase: feedDidBase,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/xrpc/app.bsky.feed.getFeedSkeleton", srv.HandleGetFeedSkeleton)
	mux.HandleFunc("/xrpc/app.bsky.feed.describeFeedGenerator", srv.HandleDescribeFeedGenerator)
	mux.HandleFunc("/.well-known/did.json", srv.HandleWellKnown)
	addr := fmt.Sprintf("0.0.0.0:%d", port)

	httpSrv := http.Server{
		Addr:    addr,
		Handler: mux,
	}

	return &Server{
		httpsrv: &httpSrv,
	}
}

func (s *Server) Run() {
	err := s.httpsrv.ListenAndServe()
	if err != nil {
		slog.Error("listen and serve", "error", err)
	}
}

func (s *Server) Stop(ctx context.Context) error {
	return s.httpsrv.Shutdown(ctx)
}

type FeedReponse struct {
	Cursor string     `json:"cursor"`
	Feed   []FeedItem `json:"feed"`
}

type FeedItem struct {
	Post        string `json:"post"`
	FeedContext string `json:"feedContext"`
}

func (s *Server) HandleGetFeedSkeleton(w http.ResponseWriter, r *http.Request) {
	slog.Info("got request for feed skeleton", "host", r.RemoteAddr)
	params := r.URL.Query()

	feed := params.Get("feed")
	if feed == "" {
		slog.Error("missing query param", "host", r.RemoteAddr)
		http.Error(w, "missing feed query param", http.StatusBadRequest)
		return
	}
	slog.Info("request for feed", "feed", feed)

	limitStr := params.Get("limit")
	limit := 50
	if limitStr != "" {
		var err error
		limit, err = strconv.Atoi(limitStr)
		if err != nil {
			slog.Error("convert limit query param", "error", err)
			http.Error(w, "invalid limit query param", http.StatusBadRequest)
			return
		}
		if limit < 1 || limit > 100 {
			limit = 50
		}
	}

	cursor := params.Get("cursor")
	usersDID, err := validateAuth(r)
	if err != nil {
		slog.Error("validate auth", "error", err)
		http.Error(w, "validate auth", http.StatusUnauthorized)
		return
	}

	resp, err := s.feeder.GetFeed(r.Context(), usersDID, feed, cursor, limit)
	if err != nil {
		slog.Error("get feed", "error", err, "feed", feed)
		http.Error(w, "error getting feed", http.StatusInternalServerError)
		return
	}

	b, err := json.Marshal(resp)
	if err != nil {
		slog.Error("marshall error", "error", err, "host", r.RemoteAddr)
		http.Error(w, "failed to encode resp", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	w.Write(b)
}

type DescribeFeedResponse struct {
	DID   string          `json:"did"`
	Feeds []FeedRespsonse `json:"feeds"`
}

type FeedRespsonse struct {
	URI string `json:"uri"`
}

func (s *Server) HandleDescribeFeedGenerator(w http.ResponseWriter, r *http.Request) {
	slog.Info("got request for describe feed", "host", r.RemoteAddr)
	resp := DescribeFeedResponse{
		DID: fmt.Sprintf("did:web:%s", s.feedHost),
		Feeds: []FeedRespsonse{
			{
				URI: fmt.Sprintf("at://%s/app.bsky.feed.generator/wills-test", s.feedDidBase),
			},
		},
	}

	b, err := json.Marshal(resp)
	if err != nil {
		http.Error(w, "failed to encode resp", http.StatusInternalServerError)
		return
	}

	w.Write(b)
}

type WellKnownResponse struct {
	Context []string           `json:"@context"`
	Id      string             `json:"id"`
	Service []WellKnownService `json:"service"`
}

type WellKnownService struct {
	Id              string `json:"id"`
	Type            string `json:"type"`
	ServiceEndpoint string `json:"serviceEndpoint"`
}

func (s *Server) HandleWellKnown(w http.ResponseWriter, r *http.Request) {
	slog.Info("got request for well known", "host", r.RemoteAddr)
	resp := WellKnownResponse{
		Context: []string{"https://www.w3.org/ns/did/v1"},
		Id:      fmt.Sprintf("did:web:%s", s.feedHost),
		Service: []WellKnownService{
			{
				Id:              "#bsky_fg",
				Type:            "BskyFeedGenerator",
				ServiceEndpoint: fmt.Sprintf("https://%s", s.feedHost),
			},
		},
	}

	b, err := json.Marshal(resp)
	if err != nil {
		http.Error(w, "failed to encode resp", http.StatusInternalServerError)
		return
	}

	w.Write(b)
}

// this extracts the DID of the user that has made the request from the JWT of the auth header
var directory = identity.DefaultDirectory()

func validateAuth(r *http.Request) (string, error) {
	headerValues := r.Header["Authorization"]
	if len(headerValues) != 1 {
		return "", fmt.Errorf("missing authorization header")
	}
	token := strings.TrimSpace(strings.Replace(headerValues[0], "Bearer ", "", 1))

	nsid := strings.Replace(r.URL.Path, "/xrpc/", "", 1)

	parsedToken, err := jwt.ParseWithClaims(token, jwt.MapClaims{}, func(token *jwt.Token) (interface{}, error) {
		did := syntax.DID(token.Claims.(jwt.MapClaims)["iss"].(string))
		identity, err := directory.LookupDID(r.Context(), did)
		if err != nil {
			return nil, fmt.Errorf("unable to resolve did %s: %s", did, err)
		}
		key, err := identity.PublicKey()
		if err != nil {
			return nil, fmt.Errorf("signing key not found for did %s: %s", did, err)
		}
		return key, nil
	})
	if err != nil {
		return "", fmt.Errorf("invalid token: %s", err)
	}

	claims := parsedToken.Claims.(jwt.MapClaims)
	if claims["lxm"] != nsid {
		return "", fmt.Errorf("bad jwt lexicon method (\"lxm\"). must match: %s", nsid)
	}
	return claims["iss"].(string), nil
}
