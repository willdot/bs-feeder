package main

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/api/atproto"
	apibsky "github.com/bluesky-social/indigo/api/bsky"
	"github.com/bluesky-social/jetstream/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/willdot/bskyfeedgen/store"
)

func TestHandlerReceivesSubscribeMessage(t *testing.T) {
	db, err := store.New(":memory:")
	require.NoError(t, err)

	handler := handler{
		store: db,
	}

	record := apibsky.FeedPost{
		Text: "/subscribe",
		Reply: &apibsky.FeedPost_ReplyRef{
			Parent: &atproto.RepoStrongRef{
				Uri: "parent-uri",
			},
		},
	}

	recordB, err := json.Marshal(record)
	require.NoError(t, err)

	event := &models.Event{
		Did: myDid,
		Commit: &models.Commit{
			Operation:  models.CommitOperationCreate,
			Collection: "app.bsky.feed.post",
			RKey:       "subscribe-post-rkey",
			Record:     recordB,
		},
	}

	// send the event twice to simulate subscribing to the same post twice, to check only
	// 1 subscription is created
	err = handler.HandleEvent(context.Background(), event)
	require.NoError(t, err)
	err = handler.HandleEvent(context.Background(), event)
	require.NoError(t, err)

	subs, err := db.GetSubscriptionsForPost("parent-uri")
	require.NoError(t, err)

	assert.Len(t, subs, 1)
	assert.Equal(t, myDid, subs[0])
}

func TestHandlerReceivesReplyToASubscribedPost(t *testing.T) {
	db, err := store.New(":memory:")
	require.NoError(t, err)

	handler := handler{
		store: db,
	}

	// add the subscription
	err = db.AddSubscriptionForPost("parent-uri", myDid, "subscribe-post-rkey")
	require.NoError(t, err)

	record := apibsky.FeedPost{
		Text: "this is a reply to a post that was subscribed to",
		Reply: &apibsky.FeedPost_ReplyRef{
			Parent: &atproto.RepoStrongRef{
				Uri: "parent-uri",
			},
		},
	}

	recordB, err := json.Marshal(record)
	require.NoError(t, err)

	event := &models.Event{
		Did: "some-random-did",
		Commit: &models.Commit{
			Operation:  models.CommitOperationCreate,
			Collection: "app.bsky.feed.post",
			RKey:       "reply-post-rkey",
			Record:     recordB,
		},
	}

	err = handler.HandleEvent(context.Background(), event)
	require.NoError(t, err)

	feed, err := db.GetUsersFeed(myDid, 9999999999999, 5)
	require.NoError(t, err)

	assert.Len(t, feed, 1)
	expectedFeedPost := store.FeedPost{
		ID:                1,
		ReplyURI:          "at://some-random-did/app.bsky.feed.post/reply-post-rkey",
		UserDID:           myDid,
		SubscribedPostURI: "parent-uri",
	}

	res := feed[0]
	// timestamps are hard to assert so check it's within a few seconds and then remove from
	// the result so the rest of the assertion can complete
	assert.WithinDuration(t, time.Now(), time.UnixMilli(res.CreatedAt), time.Second)
	res.CreatedAt = 0

	assert.Equal(t, expectedFeedPost, res)
}

func TestHandlerReceivesDeleteEvent(t *testing.T) {
	db, err := store.New(":memory:")
	require.NoError(t, err)

	handler := handler{
		store: db,
	}

	// add the subscription
	err = db.AddSubscriptionForPost("parent-uri", myDid, "subscribe-post-rkey")
	require.NoError(t, err)
	// add in some feed posts
	feedPost1 := store.FeedPost{
		ReplyURI:          "at://some-random-did-1/app.bsky.feed.post/reply-post-rkey",
		UserDID:           myDid,
		SubscribedPostURI: "parent-uri",
	}
	feedPost2 := store.FeedPost{
		ReplyURI:          "at://some-random-did-2/app.bsky.feed.post/reply-post-rkey",
		UserDID:           myDid,
		SubscribedPostURI: "parent-uri",
	}
	err = db.AddFeedPost(feedPost1)
	require.NoError(t, err)
	err = db.AddFeedPost(feedPost2)
	require.NoError(t, err)
	// add a feed post for a different subscribed post
	feedPost3 := store.FeedPost{
		ReplyURI:          "at://some-random-did-3/app.bsky.feed.post/reply-post-rkey",
		UserDID:           myDid,
		SubscribedPostURI: "different-parent-uri",
	}
	err = db.AddFeedPost(feedPost3)
	require.NoError(t, err)

	event := &models.Event{
		Did: myDid,
		Commit: &models.Commit{
			Operation:  models.CommitOperationDelete,
			Collection: "app.bsky.feed.post",
			RKey:       "subscribe-post-rkey",
		},
	}

	err = handler.HandleEvent(context.Background(), event)
	require.NoError(t, err)

	feed, err := db.GetUsersFeed(myDid, 9999999999999, 5)
	require.NoError(t, err)

	assert.Len(t, feed, 1)
	feedPost3.ID = 3
	assert.Equal(t, feedPost3, feed[0])
}
