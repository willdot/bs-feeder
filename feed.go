package main

import (
	"context"
	"sync"
)

type FeedGenerator struct {
	mu    sync.Mutex
	feeds []string
}

// TODO: pass in feeds or something
func NewFeedGenerator() *FeedGenerator {
	return &FeedGenerator{}
}

func (f *FeedGenerator) GetFeed(ctx context.Context, feed, cursor string, limit int) (*FeedReponse, error) {
	// TODO: get something from a database instead
	// resp := &FeedReponse{
	// 	Feed: []FeedItem{
	// 		{
	// 			Post:        "at://did:plc:dadhhalkfcq3gucaq25hjqon/app.bsky.feed.post/3l7j5ma2si42r",
	// 			FeedContext: "this is some additional context",
	// 		},
	// 	},
	// 	Cursor: "",
	// }
	f.mu.Lock()
	defer f.mu.Unlock()
	feedItems := make([]FeedItem, 0, len(f.feeds))
	for _, feed := range f.feeds {
		feedItems = append(feedItems, FeedItem{
			Post: feed,
		})
	}

	resp := &FeedReponse{
		Feed:   feedItems,
		Cursor: "",
	}

	return resp, nil
}

func (f *FeedGenerator) AddToFeed(postURI string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	// TODO: store this in DB instead
	f.feeds = append(f.feeds, postURI)
}
