package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/bugsnag/bugsnag-go/v2"
)

type FeedGenerator struct {
	db *sql.DB
}

func NewFeedGenerator(db *sql.DB) *FeedGenerator {
	return &FeedGenerator{
		db: db,
	}
}

func (f *FeedGenerator) GetFeed(ctx context.Context, userDID, feed, cursor string, limit int) (FeedReponse, error) {
	resp := FeedReponse{
		Feed: make([]FeedItem, 0, 0),
	}

	usersFeed, err := getUsersFeedItems(f.db, userDID)
	if err != nil {
		return resp, fmt.Errorf("get users feed items from DB: %w", err)
	}

	feedItems := make([]FeedItem, 0, len(usersFeed))
	for _, post := range usersFeed {
		feedItems = append(feedItems, FeedItem{
			Post: post.URI,
		})
	}

	resp.Feed = feedItems
	resp.Cursor = ""

	return resp, nil
}

func (f *FeedGenerator) AddToFeedPosts(usersDids []string, parentURI, postURI string) {
	for _, did := range usersDids {
		feedItem := feedItem{
			URI:       postURI,
			UserDID:   did,
			parentURI: parentURI,
		}
		err := addFeedItem(context.Background(), f.db, feedItem)
		if err != nil {
			slog.Error("add users feed item", "error", err, "did", did, "uri", postURI)
			bugsnag.Notify(err)
			continue
		}
	}
}
