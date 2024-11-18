package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/bugsnag/bugsnag-go/v2"
	"github.com/willdot/bskyfeedgen/store"
)

type FeedGenerator struct {
	store *store.Store
}

func NewFeedGenerator(store *store.Store) *FeedGenerator {
	return &FeedGenerator{
		store: store,
	}
}

func (f *FeedGenerator) GetFeed(ctx context.Context, userDID, feed, cursor string, limit int) (FeedReponse, error) {
	resp := FeedReponse{
		Feed: make([]FeedItem, 0, 0),
	}

	usersFeed, err := f.store.GetUsersFeedItems(userDID)
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

func (f *FeedGenerator) AddToFeedPosts(usersDids []string, subscribedPostURI, replyPostURI string) {
	for _, did := range usersDids {
		feedItem := store.FeedItem{
			ReplyURI:          replyPostURI,
			UserDID:           did,
			SubscribedPostURI: subscribedPostURI,
		}
		err := f.store.AddFeedItem(feedItem)
		if err != nil {
			slog.Error("add users feed item", "error", err, "did", did, "reply post URI", replyPostURI)
			bugsnag.Notify(err)
			continue
		}
	}
}
