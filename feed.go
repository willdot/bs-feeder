package main

import "context"

type FeedGenerator struct {
	feeds []string
}

// TODO: pass in feeds or something
func NewFeedGenerator() *FeedGenerator {
	return &FeedGenerator{}
}

func (f *FeedGenerator) GetFeed(ctx context.Context, feed, cursor string, limit int) (*FeedReponse, error) {
	// TODO: get something from a database
	resp := &FeedReponse{
		Feed: []FeedItme{
			{
				Post: "at://did:plc:dadhhalkfcq3gucaq25hjqon/app.bsky.feed.post/3l7j5ma2si42r",
			},
		},
		Cursor: "",
	}
	return resp, nil
}
