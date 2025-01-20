package main

import (
	"context"
	_ "embed"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/bluesky-social/indigo/xrpc"
	"github.com/willdot/bskyfeedgen/store"
)

type Feeder interface {
	GetFeed(ctx context.Context, userDID, feed, cursor string, limit int) (FeedReponse, error)
	GetSubscriptionsForUser(ctx context.Context, userDID string) ([]store.Subscription, error)
	DeleteSubscriptionBySubRKeyAndUser(userDID, rkey string) error
	DeleteFeedPostsForSubscribedPostURIandUserDID(subscribedPostURI, userDID string) error
	GetSubscriptionURIByRKeyAndUserDID(userDID, rkey string) (string, error)
}

type BookmarkStore interface {
	CreateBookmark(postRKey, postURI, postATURI, authorDID, authorHandle, userDID, content string) error
	GetBookmarksForUser(userDID string) ([]store.Bookmark, error)
	DeleteBookmark(postRKey, userDID string) error
	GetBookmarkByRKeyForUser(rkey, userDID string) (*store.Bookmark, error)
}

type Server struct {
	httpsrv       *http.Server
	feeder        Feeder
	feedHost      string
	feedDidBase   string
	bookmarkStore BookmarkStore
	xrpcClient    *xrpc.Client
}

func NewServer(port int, feeder Feeder, feedHost, feedDidBase string, bookmarkStore BookmarkStore) *Server {
	srv := &Server{
		feeder:        feeder,
		feedHost:      feedHost,
		feedDidBase:   feedDidBase,
		bookmarkStore: bookmarkStore,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/public/styles.css", serveCSS)
	mux.HandleFunc("/xrpc/app.bsky.feed.getFeedSkeleton", srv.HandleGetFeedSkeleton)
	mux.HandleFunc("/xrpc/app.bsky.feed.describeFeedGenerator", srv.HandleDescribeFeedGenerator)
	mux.HandleFunc("/.well-known/did.json", srv.HandleWellKnown)

	mux.HandleFunc("/", srv.authMiddleware(srv.HandleSubscriptions))
	mux.HandleFunc("/login", srv.HandleLogin)
	mux.HandleFunc("GET /subscriptions", srv.HandleSubscriptions)
	mux.HandleFunc("DELETE /sub/{id}", srv.HandleDeleteSubscription)
	mux.HandleFunc("GET /bookmarks", srv.HandleGetBookmarks)
	mux.HandleFunc("POST /bookmarks", srv.HandleAddBookmark)
	mux.HandleFunc("DELETE /bookmarks/{rkey}", srv.HandleDeleteBookmark)

	addr := fmt.Sprintf("0.0.0.0:%d", port)

	srv.httpsrv = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	srv.xrpcClient = &xrpc.Client{
		// Client: http.DefaultClient,
		Host: "https://public.api.bsky.app",
	}

	return srv
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

//go:embed public/styles.css
var cssFile []byte

func serveCSS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/css; charset=utf-8")
	w.Write(cssFile)
}

func getUsersDidFromRequest(r *http.Request) (string, error) {
	didCookie, err := r.Cookie(didCookieName)
	if err != nil {
		return "", err
	}
	if didCookie == nil {
		return "", fmt.Errorf("missing did cookie")
	}

	return didCookie.Value, nil
}
