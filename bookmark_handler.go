package main

import (
	"encoding/json"
	"errors"
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
	usersDid, err := getUsersDidFromRequestCookie(r)
	if err != nil {
		slog.Error("getting users did from request", "error", err)
		_ = frontend.Login("", "").Render(r.Context(), w)
		return
	}

	postURI := r.FormValue("uri")
	postURI = strings.TrimSuffix(postURI, "/")

	atPostURI, err := convertPostURIToAtValidURI(postURI)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if atPostURI == "at://" {
		http.Error(w, "invalid post URI - contains invalid user handle", http.StatusBadRequest)
		return
	}

	uriSplit := strings.Split(atPostURI, "/")
	rkey := uriSplit[len(uriSplit)-1]

	postResp, err := bsky.FeedGetPosts(r.Context(), s.xrpcClient, []string{atPostURI})
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
	postBytes, err := post.Record.MarshalJSON()
	if err != nil {
		slog.Error("marshal post record", "error", err)
		http.Error(w, "decode the post from Bluesky", http.StatusInternalServerError)
		return
	}

	var postRecord apibsky.FeedPost
	if err := json.Unmarshal(postBytes, &postRecord); err != nil {
		slog.Error("unmarshal post record", "error", err)
		http.Error(w, "decode the post from Bluesky", http.StatusInternalServerError)
		return
	}

	content := postRecord.Text
	if len(content) > 75 {
		content = fmt.Sprintf("%s...", content[:75])
	}

	err = s.bookmarkStore.CreateBookmark(rkey, postURI, atPostURI, post.Author.Did, post.Author.Handle, usersDid, content)
	if err != nil {
		if errors.Is(err, store.ErrBookmarkAlreadyExists) {
			return
		}
		slog.Error("create bookmark", "error", err)
		http.Error(w, "failed to create bookmark", http.StatusInternalServerError)
		return
	}

	bookmark := store.Bookmark{
		PostRKey:     rkey,
		PostURI:      postURI,
		PostATURI:    atPostURI,
		AuthorDID:    post.Author.Did,
		AuthorHandle: post.Author.Handle,
		UserDID:      usersDid,
		Content:      content,
	}

	_ = frontend.NewBookmarkRow(bookmark).Render(r.Context(), w)
}

func convertPostURIToAtValidURI(input string) (string, error) {
	input = strings.TrimPrefix(input, "https://bsky.app/profile/")
	b := strings.Split(input, "/")

	did, err := resolveHandle(b[0])
	if err != nil {
		slog.Error("error resolving handle", "error", err)
		return "", fmt.Errorf("error resolving handle")
	}

	input = strings.ReplaceAll(input, b[0], did)
	input = strings.ReplaceAll(input, "https://bsky.app/profile/", "")

	return fmt.Sprintf("at://%s", strings.ReplaceAll(input, "post", "app.bsky.feed.post")), nil
}

func (s *Server) HandleDeleteBookmark(w http.ResponseWriter, r *http.Request) {
	rKey := r.PathValue("rkey")

	usersDid, err := getUsersDidFromRequestCookie(r)
	if err != nil {
		slog.Error("getting users did from request", "error", err)
		_ = frontend.Login("", "").Render(r.Context(), w)
		return
	}

	bookmark, err := s.bookmarkStore.GetBookmarkByRKeyForUser(rKey, usersDid)
	if err != nil {
		slog.Error("getting bookmark by rkey and users did", "error", err)
		http.Error(w, "getting bookmark to delete", http.StatusInternalServerError)
		return
	}

	err = s.bookmarkStore.DeleteFeedPostsForBookmarkedPostURIandUserDID(bookmark.PostATURI, usersDid)
	if err != nil {
		slog.Error("deleting feed items for bookmark", "error", err)
		http.Error(w, "deleting feed items for bookmark", http.StatusInternalServerError)
		return
	}

	err = s.bookmarkStore.DeleteBookmark(rKey, usersDid)
	if err != nil {
		slog.Error("delete bookmark", "error", err)
		http.Error(w, "failed to delete bookmark", http.StatusInternalServerError)
		// TODO: what to return to client
		return
	}

	w.WriteHeader(http.StatusAccepted)
	_, _ = w.Write([]byte("{}"))
}

func (s *Server) HandleGetBookmarks(w http.ResponseWriter, r *http.Request) {
	usersDid, err := getUsersDidFromRequestCookie(r)
	if err != nil {
		slog.Error("getting users did from request", "error", err)
		_ = frontend.Login("", "").Render(r.Context(), w)
		return
	}

	bookmarks, err := s.bookmarkStore.GetBookmarksForUser(usersDid)
	if err != nil {
		slog.Error("error getting bookmarks for user", "error", err)
		_ = frontend.Bookmarks(nil).Render(r.Context(), w)
		return
	}

	resp := make([]store.Bookmark, 0, len(bookmarks))
	resp = append(resp, bookmarks...)

	_ = frontend.Bookmarks(resp).Render(r.Context(), w)
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
