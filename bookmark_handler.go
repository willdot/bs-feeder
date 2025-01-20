package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/bluesky-social/indigo/api/bsky"
	apibsky "github.com/bluesky-social/indigo/api/bsky"
	"github.com/willdot/bskyfeedgen/frontend"
	"github.com/willdot/bskyfeedgen/store"
)

func (s *Server) HandleAddBookmark(w http.ResponseWriter, r *http.Request) {
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

	postURI := r.FormValue("uri")

	a := strings.TrimPrefix(postURI, "https://bsky.app/profile/")
	b := strings.Split(a, "/")

	did, err := resolveHandle(b[0])
	if err != nil {
		slog.Error("error revolving handle", "error", err)
		http.Error(w, "resolving handle", http.StatusInternalServerError)
		return
	}
	postURI = strings.ReplaceAll(postURI, b[0], did)
	postURI = strings.ReplaceAll(postURI, "https://bsky.app/profile/", "")

	sanitizedPostURI := fmt.Sprintf("at://%s", strings.ReplaceAll(postURI, "post", "app.bsky.feed.post"))

	uriSplit := strings.Split(sanitizedPostURI, "/")
	rkey := uriSplit[len(uriSplit)-1]

	postResp, err := bsky.FeedGetPosts(r.Context(), s.xrpcClient, []string{sanitizedPostURI})
	if err != nil {
		slog.Error("error getting post details from Bsky", "error", err)
		http.Error(w, "error fetching post details from Bluesky", http.StatusInternalServerError)
		return
	}

	if postResp == nil || len(postResp.Posts) != 1 {
		http.Error(w, "post not found", http.StatusNotFound)
		return
	}

	post := postResp.Posts[0]

	slog.Info("record", "val", post)

	postB, err := post.Record.MarshalJSON()
	if err != nil {
		slog.Error("marshal post record", "error", err)
		http.Error(w, "decode the post from Bluesky", http.StatusInternalServerError)
		return
	}

	var postRecord apibsky.FeedPost
	if err := json.Unmarshal(postB, &postRecord); err != nil {
		slog.Error("unmarshal post record", "error", err)
		http.Error(w, "decode the post from Bluesky", http.StatusInternalServerError)
		return
	}

	content := postRecord.Text
	if len(content) > 75 {
		content = fmt.Sprintf("%s...", content[:75])
	}

	err = s.bookmarkStore.CreateBookmark(rkey, postURI, post.Author.Did, post.Author.Handle, usersDid, content)
	if err != nil {
		slog.Error("create bookmark", "error", err)
		http.Error(w, "failed to create bookmark", http.StatusInternalServerError)
		return
	}

	bookmark := store.Bookmark{
		PostRKey:     rkey,
		PostURI:      postURI,
		AuthorDID:    post.Author.Did,
		AuthorHandle: post.Author.Handle,
		UserDID:      usersDid,
		Content:      content,
	}

	frontend.NewBookmarkRow(bookmark).Render(r.Context(), w)
}

func (s *Server) HandleDeleteBookmark(w http.ResponseWriter, r *http.Request) {
	rKey := r.PathValue("rkey")

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

	err = s.bookmarkStore.DeleteBookmark(rKey, usersDid)
	if err != nil {
		slog.Error("delete bookmark", "error", err)
		http.Error(w, "failed to delete bookmark", http.StatusInternalServerError)
		// TODO: what to return to client
		return
	}

	// TODO: delete feed items too

	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte("{}"))
}

func (s *Server) HandleGetBookmarks(w http.ResponseWriter, r *http.Request) {
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

	bookmarks, err := s.bookmarkStore.GetBookmarksForUser(usersDid)
	if err != nil {
		slog.Error("error getting bookmarks for user", "error", err)
		frontend.Subscriptions("failed to get bookmarks", nil).Render(r.Context(), w)
		return
	}

	resp := make([]store.Bookmark, 0, len(bookmarks))
	for _, bookmark := range bookmarks {
		// splitStr := strings.Split(sub.SubscribedPostURI, "/")

		// if len(splitStr) != 5 {
		// 	slog.Error("subscription URI was not expected - expected to have 5 strings after spliting by /", "uri", sub.SubscribedPostURI)
		// 	continue
		// }

		// did := splitStr[2]

		// handle, err := resolveDid(did)
		// if err != nil {
		// 	slog.Error("resolving did", "error", err, "did", did)
		// 	handle = did
		// }

		// uri := fmt.Sprintf("https://bsky.app/profile/%s/post/%s", handle, splitStr[4])
		// sub.SubscribedPostURI = uri
		resp = append(resp, bookmark)
	}

	frontend.Bookmarks("", resp).Render(r.Context(), w)
}

func resolveHandle(handle string) (string, error) {
	params := url.Values{
		"handle": []string{handle},
	}
	reqUrl := "https://public.api.bsky.app/xrpc/com.atproto.identity.resolveHandle?" + params.Encode()

	resp, err := http.DefaultClient.Get(reqUrl)
	if err != nil {
		return "", fmt.Errorf("make http request: %w", err)
	}

	defer resp.Body.Close()

	type did struct {
		Did string
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response body: %w", err)
	}

	var resDid did
	err = json.Unmarshal(b, &resDid)
	if err != nil {
		return "", fmt.Errorf("unmarshal response: %w", err)
	}

	return resDid.Did, nil
}
