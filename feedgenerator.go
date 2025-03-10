package main

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/willdot/bskyfeedgen/store"
)

type repliesStore interface {
	GetUsersReplies(usersDID string, cursor int64, limit int) ([]store.ReplyPost, error)
	GetBookmarksForUserWithPaging(userDID string, cursor int64, limit int) ([]store.Bookmark, error)
	AddRepliedPost(replyPost store.ReplyPost) error
}

type FeedGenerator struct {
	store repliesStore
}

func NewFeedGenerator(store repliesStore) *FeedGenerator {
	return &FeedGenerator{
		store: store,
	}
}

func (f *FeedGenerator) GetFeed(ctx context.Context, userDID, feed, cursor string, limit int) (FeedReponse, error) {
	switch {
	case strings.Contains(feed, "bookmark-replies"):
		return f.getBookmarkRepliesFeed(ctx, userDID, cursor, limit)
	case strings.Contains(feed, "bookmarks"):
		return f.getBookmarksFeed(ctx, userDID, cursor, limit)

	default:
		return FeedReponse{
			Feed: make([]FeedItem, 0),
		}, fmt.Errorf("invalid feed requested")
	}
}

func (f *FeedGenerator) getBookmarkRepliesFeed(ctx context.Context, userDID, cursor string, limit int) (FeedReponse, error) {
	resp := FeedReponse{
		Feed: make([]FeedItem, 0),
	}

	cursorInt, err := strconv.Atoi(cursor)
	if err != nil && cursor != "" {
		slog.Error("convert cursor to int", "error", err, "cursor value", cursor)
	}
	if cursorInt == 0 {
		// if no cursor provided use a date waaaaay in the future to start the less than query
		cursorInt = 9999999999999
	}

	usersReplies, err := f.store.GetUsersReplies(userDID, int64(cursorInt), limit)
	if err != nil {
		return resp, fmt.Errorf("get users replies from DB: %w", err)
	}

	feedItems := make([]FeedItem, 0, len(usersReplies))
	for _, post := range usersReplies {
		feedItems = append(feedItems, FeedItem{
			Post: post.ReplyURI,
		})
	}

	resp.Feed = feedItems

	// only set the return cursor if there was a record returned and that the len of records
	// being returned is the same as the limit
	if len(usersReplies) > 0 && len(usersReplies) == limit {
		lastFeedItem := usersReplies[len(usersReplies)-1]
		resp.Cursor = fmt.Sprintf("%d", lastFeedItem.CreatedAt)
	}
	return resp, nil
}

func (f *FeedGenerator) getBookmarksFeed(ctx context.Context, userDID, cursor string, limit int) (FeedReponse, error) {
	resp := FeedReponse{
		Feed: make([]FeedItem, 0),
	}

	cursorInt, err := strconv.Atoi(cursor)
	if err != nil && cursor != "" {
		slog.Error("convert cursor to int", "error", err, "cursor value", cursor)
	}
	if cursorInt == 0 {
		// if no cursor provided use a date waaaaay in the future to start the less than query
		cursorInt = 9999999999999
	}

	usersBookmarks, err := f.store.GetBookmarksForUserWithPaging(userDID, int64(cursorInt), limit)
	if err != nil {
		return resp, fmt.Errorf("get users bookmarks from DB: %w", err)
	}

	feedItems := make([]FeedItem, 0, len(usersBookmarks))
	for _, bookmark := range usersBookmarks {
		feedItems = append(feedItems, FeedItem{
			Post: bookmark.PostATURI,
		})
	}

	resp.Feed = feedItems

	// only set the return cursor if there was a record returned and that the len of records
	// being returned is the same as the limit
	if len(usersBookmarks) > 0 && len(usersBookmarks) == limit {
		lastFeedItem := usersBookmarks[len(usersBookmarks)-1]
		resp.Cursor = fmt.Sprintf("%d", lastFeedItem.CreatedAt)
	}
	return resp, nil
}
