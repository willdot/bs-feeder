package main

import (
	"context"
	"sync"
)

type FeedGenerator struct {
	mu    sync.Mutex
	posts map[string][]string
}

func NewFeedGenerator() *FeedGenerator {
	return &FeedGenerator{
		posts: make(map[string][]string),
	}
}

func (f *FeedGenerator) GetFeed(ctx context.Context, userDID, feed, cursor string, limit int) (*FeedReponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	usersFeed, ok := f.posts[userDID]
	if !ok {
		return nil, nil
	}

	feedItems := make([]FeedItem, 0, len(f.posts))
	for _, post := range usersFeed {
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

func (f *FeedGenerator) AddToFeedPosts(usersDid, postURI string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	// TODO: store this in DB instead
	usersPosts, ok := f.posts[usersDid]
	if !ok {
		usersPosts = make([]string, 0, 1)
	}

	usersPosts = append(usersPosts, postURI)
	f.posts[usersDid] = usersPosts
}
