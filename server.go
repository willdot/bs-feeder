package main

import (
	"context"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/bluesky-social/indigo/xrpc"
	"github.com/gorilla/sessions"
	oauth "github.com/haileyok/atproto-oauth-golang"
	oauthhelpers "github.com/haileyok/atproto-oauth-golang/helpers"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/willdot/bskyfeedgen/store"
)

const (
	serverBase = "https://bs-feeder-staging.up.railway.app"
)

type Feeder interface {
	GetFeed(ctx context.Context, userDID, feed, cursor string, limit int) (FeedReponse, error)
}

type BookmarkStore interface {
	CreateBookmark(postRKey, postURI, postATURI, authorDID, authorHandle, userDID, content string) error
	GetBookmarksForUser(userDID string) ([]store.Bookmark, error)
	DeleteBookmark(postRKey, userDID string) error
	GetBookmarkByRKeyForUser(rkey, userDID string) (*store.Bookmark, error)
	DeleteFeedPostsForBookmarkedPostURIandUserDID(subscribedPostURI, userDID string) error
}

type Server struct {
	httpsrv       *http.Server
	feeder        Feeder
	feedHost      string
	feedDidBase   string
	bookmarkStore BookmarkStore
	xrpcClient    *xrpc.Client
	jwks          *JWKS
	oauthClient   *oauth.Client
	sessionStore  *sessions.CookieStore
}

type JWKS struct {
	public  []byte
	private jwk.Key
}

func NewServer(port int, feeder Feeder, feedHost, feedDidBase string, bookmarkStore BookmarkStore) (*Server, error) {
	jwks, err := getJWKS()
	if err != nil {
		return nil, fmt.Errorf("create public JWKS: %w", err)
	}

	oauthClient, err := createOauthClient(jwks)
	if err != nil {
		return nil, fmt.Errorf("create oauth client: %w", err)
	}

	sessionStore := sessions.NewCookieStore([]byte(os.Getenv("SESSION_KEY")))

	srv := &Server{
		feeder:        feeder,
		feedHost:      feedHost,
		feedDidBase:   feedDidBase,
		bookmarkStore: bookmarkStore,
		jwks:          jwks,
		oauthClient:   oauthClient,
		sessionStore:  sessionStore,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/public/styles.css", serveCSS)
	mux.HandleFunc("/xrpc/app.bsky.feed.getFeedSkeleton", srv.HandleGetFeedSkeleton)
	mux.HandleFunc("/xrpc/app.bsky.feed.describeFeedGenerator", srv.HandleDescribeFeedGenerator)
	mux.HandleFunc("/.well-known/did.json", srv.HandleWellKnown)
	mux.HandleFunc("/client-metadata.json", serveClientMetadata)
	mux.HandleFunc("/jwks.json", srv.serverJwks)
	mux.HandleFunc("/oauth-callback", srv.handleOauthCallback)

	mux.HandleFunc("/test", srv.HandleTest)

	mux.HandleFunc("/", srv.authMiddleware(srv.HandleGetBookmarks))
	mux.HandleFunc("/login", srv.HandleLogin)
	// mux.HandleFunc("/sign-out", srv.HandleSignOut)
	mux.HandleFunc("GET /bookmarks", srv.authMiddleware(srv.HandleGetBookmarks))
	mux.HandleFunc("POST /bookmarks", srv.authMiddleware(srv.HandleAddBookmark))
	mux.HandleFunc("DELETE /bookmarks/{rkey}", srv.authMiddleware(srv.HandleDeleteBookmark))

	addr := fmt.Sprintf("0.0.0.0:%d", port)

	srv.httpsrv = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	srv.xrpcClient = &xrpc.Client{
		// Client: http.DefaultClient,
		Host: "https://public.api.bsky.app",
	}

	return srv, nil
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
	_, _ = w.Write(cssFile)
}

func getUsersDidFromRequestCookie(r *http.Request) (string, error) {
	didCookie, err := r.Cookie(didCookieName)
	if err != nil {
		return "", err
	}
	if didCookie == nil {
		return "", fmt.Errorf("missing did cookie")
	}

	return didCookie.Value, nil
}

func getJWKS() (*JWKS, error) {
	jwksB64 := os.Getenv("PRIVATEJWKS")
	if jwksB64 == "" {
		return nil, fmt.Errorf("PRIVATEJWKS env not set")
	}

	jwksB, err := base64.StdEncoding.DecodeString(jwksB64)
	if err != nil {
		return nil, fmt.Errorf("decode jwks env: %w", err)
	}

	k, err := oauthhelpers.ParseJWKFromBytes([]byte(jwksB))
	if err != nil {
		return nil, fmt.Errorf("parse JWK from bytes: %w", err)
	}

	pubkey, err := k.PublicKey()
	if err != nil {
		return nil, fmt.Errorf("get public key from JWKS: %w", err)
	}

	resp := oauthhelpers.CreateJwksResponseObject(pubkey)
	b, err := json.Marshal(resp)
	if err != nil {
		return nil, fmt.Errorf("marshal public JWKS: %w", err)
	}

	return &JWKS{
		public:  b,
		private: k,
	}, nil
}

func createOauthClient(jwks *JWKS) (*oauth.Client, error) {
	return oauth.NewClient(oauth.ClientArgs{
		ClientJwk:   jwks.private,
		ClientId:    fmt.Sprintf("%s/client-metadata.json", serverBase),
		RedirectUri: fmt.Sprintf("%s/oauth-callback", serverBase),
	})
}

//did:plc:dadhhalkfcq3gucaq25hjqon
