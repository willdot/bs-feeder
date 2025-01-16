package main

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/willdot/bskyfeedgen/frontend"
	"github.com/willdot/bskyfeedgen/store"
)

const (
	bskyBaseURL = "https://bsky.social/xrpc"
)

type Feeder interface {
	GetFeed(ctx context.Context, userDID, feed, cursor string, limit int) (FeedReponse, error)
	GetSubscriptionsForUser(ctx context.Context, userDID string) ([]store.Subscription, error)
}

type Server struct {
	httpsrv      *http.Server
	feeder       Feeder
	feedHost     string
	feedDidBase  string
	jwtSecretKey string
}

func NewServer(port int, feeder Feeder, feedHost, feedDidBase string) *Server {
	secretKey := os.Getenv("JWT_SECRET")
	if secretKey == "" {
		secretKey = "TEST_KEY"
	}

	srv := &Server{
		feeder:       feeder,
		feedHost:     feedHost,
		feedDidBase:  feedDidBase,
		jwtSecretKey: secretKey,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/public/styles.css", serveCSS)
	mux.HandleFunc("/xrpc/app.bsky.feed.getFeedSkeleton", srv.HandleGetFeedSkeleton)
	mux.HandleFunc("/xrpc/app.bsky.feed.describeFeedGenerator", srv.HandleDescribeFeedGenerator)
	mux.HandleFunc("/.well-known/did.json", srv.HandleWellKnown)

	mux.HandleFunc("/", srv.authMiddleware(srv.HandleSubscriptions))
	mux.HandleFunc("/login", srv.HandleLogin)

	// mux.HandleFunc("/subscriptions", srv.HandleSubscriptions)

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

//go:embed public/styles.css
var cssFile []byte

func serveCSS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/css; charset=utf-8")
	w.Write(cssFile)
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
		frontend.Subscriptions("failed to get subscriptions", []string{}).Render(r.Context(), w)
		return
	}

	sanitizedURIs := make([]string, 0, len(subs))
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

		uri := fmt.Sprintf("https://bsky.app/profile/%s/post/%s", handle, splitStr[4])
		sanitizedURIs = append(sanitizedURIs, uri)
	}

	frontend.Subscriptions("", sanitizedURIs).Render(r.Context(), w)
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
