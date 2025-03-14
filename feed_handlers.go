package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
)

type FeedReponse struct {
	Cursor string     `json:"cursor"`
	Feed   []FeedItem `json:"feed"`
}

type FeedItem struct {
	Post        string `json:"post"`
	FeedContext string `json:"feedContext"`
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

func (s *Server) HandleGetFeedSkeleton(w http.ResponseWriter, r *http.Request) {
	slog.Info("got request for feed skeleton", "host", r.RemoteAddr)
	params := r.URL.Query()

	feed := params.Get("feed")
	if feed == "" {
		slog.Error("missing feed query param", "host", r.RemoteAddr)
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
	usersDID, err := getRequestUserDID(r)
	if err != nil {
		slog.Error("validate auth", "error", err)
		http.Error(w, "validate auth", http.StatusUnauthorized)
		return
	}
	if usersDID == "" {
		slog.Error("missing users DID from request")
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

	_, _ = w.Write(b)
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
				URI: fmt.Sprintf("at://%s/app.bsky.feed.generator/bookmark-replies", s.feedDidBase),
			},
			{
				URI: fmt.Sprintf("at://%s/app.bsky.feed.generator/bookmarks", s.feedDidBase),
			},
		},
	}

	b, err := json.Marshal(resp)
	if err != nil {
		http.Error(w, "failed to encode resp", http.StatusInternalServerError)
		return
	}

	_, _ = w.Write(b)
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

	_, _ = w.Write(b)
}
