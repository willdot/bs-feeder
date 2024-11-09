package main

import (
	"context"
	"sync"
)

type FeedGenerator struct {
	mu    sync.Mutex
	posts map[string]struct{}
}

func NewFeedGenerator() *FeedGenerator {
	return &FeedGenerator{
		posts: make(map[string]struct{}),
	}
}

func (f *FeedGenerator) GetFeed(ctx context.Context, feed, cursor string, limit int) (*FeedReponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	feedItems := make([]FeedItem, 0, len(f.posts))
	for post := range f.posts {
		feedItems = append(feedItems, FeedItem{
			Post: post,
		})
	}

	resp := &FeedReponse{
		Feed:   feedItems,
		Cursor: "",
	}

	return resp, nil
}

func (f *FeedGenerator) AddToFeedPosts(postURI string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	// TODO: store this in DB instead
	f.posts[postURI] = struct{}{}
}
