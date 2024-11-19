package main

import (
	"context"
	"fmt"

	"github.com/willdot/bskyfeedgen/store"
)

type feedStore interface {
	GetUsersFeed(usersDID string) ([]store.FeedPost, error)
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

	usersFeed, err := f.store.GetUsersFeed(userDID)
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
	resp.Cursor = ""

	return resp, nil
}
