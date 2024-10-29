package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
)

type Feeder interface {
	GetFeed(ctx context.Context, feed, cursor string, limit int) (*FeedReponse, error)
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
	Feed   []FeedItme `json:"feed"`
}

type FeedItme struct {
	Post string `json:"post"`
}

func (s *Server) HandleGetFeedSkeleton(w http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()

	feed := params.Get("feed")
	if feed == "" {
		http.Error(w, "missing feed query param", http.StatusBadRequest)
		return
	}

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
	slog.Info("cursor", "val", cursor)

	// TODO: get things from DB and return it

	resp, err := s.feeder.GetFeed(r.Context(), feed, cursor, limit)
	if err != nil {
		slog.Error("get feed", "error", err, "feed", feed)
		http.Error(w, "error getting feed", http.StatusInternalServerError)
		return
	}

	b, err := json.Marshal(resp)
	if err != nil {
		http.Error(w, "failed to encode resp", http.StatusInternalServerError)
		return
	}

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
	resp := DescribeFeedResponse{
		DID: fmt.Sprintf("did:web:%s", s.feedHost),
		Feeds: []FeedRespsonse{
			{
				URI: fmt.Sprintf("at://%s.app.bsky.feed.generator/wills-test", s.feedDidBase),
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

// "@context": ["https://www.w3.org/ns/did/v1"],
//
//	"id": FEED_DID,
//	"service":[{
//		"id": "#bsky_fg",
//		"type": "BskyFeedGenerator",
//		"serviceEndpoint": f"https://{FEED_HOSTNAME}"
//	}]

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
	resp := WellKnownResponse{
		Context: []string{"https://www.w3.org/ns/did/v1"},
		Id:      fmt.Sprintf("did:web:%s", s.feedHost),
		Service: []WellKnownService{
			WellKnownService{
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
