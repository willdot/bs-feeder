package main

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"

	"github.com/willdot/bskyfeedgen/store"
)

type feedStore interface {
	GetUsersFeed(usersDID string, cursor int64, limit int) ([]store.FeedPost, error)
}

type FeedGenerator struct {
	store feedStore
}

func NewFeedGenerator(store feedStore) *FeedGenerator {
	return &FeedGenerator{
		store: store,
	}
}

func (f *FeedGenerator) GetFeed(ctx context.Context, userDID, feed, cursor string, limit int) (FeedReponse, error) {
	resp := FeedReponse{
		Feed: make([]FeedItem, 0, 0),
	}

	cursorInt, err := strconv.Atoi(cursor)
	if err != nil && cursor != "" {
		slog.Error("convert cursor to int", "error", err, "cursor value", cursor)
	}
	if cursorInt == 0 {
		// if no cursor provided use a date waaaaay in the future to start the less than query
		cursorInt = 9999999999999
	}

	usersFeed, err := f.store.GetUsersFeed(userDID, int64(cursorInt), limit)
	if err != nil {
		return resp, fmt.Errorf("get users feed items from DB: %w", err)
	}

	feedItems := make([]FeedItem, 0, len(usersFeed))
	for _, post := range usersFeed {
		feedItems = append(feedItems, FeedItem{
			Post: post.ReplyURI,
		})
	}

	resp.Feed = feedItems

	// only set the return cursor if there was a record returned and that the len of records
	// being returned is the same as the limit
	if len(usersFeed) > 0 && len(usersFeed) == limit {
		lastFeedItem := usersFeed[len(usersFeed)-1]
		resp.Cursor = fmt.Sprintf("%d", lastFeedItem.CreatedAt)
	}
	return resp, nil
}
