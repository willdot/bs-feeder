package main

import (
	"context"
	"encoding/json"
	"testing"

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
		Did: myDid,
		Commit: &models.Commit{
			Operation:  models.CommitOperationCreate,
			Collection: "app.bsky.feed.post",
			RKey:       "reply-post-rkey",
			Record:     recordB,
		},
	}

	// send the event twice to simulate subscribing to the same post twice, to check only
	// 1 subscription is created
	err = handler.HandleEvent(context.Background(), event)
	require.NoError(t, err)
	err = handler.HandleEvent(context.Background(), event)
	require.NoError(t, err)

	subs, err := db.GetSubscriptionsForPost("some-uri")
	require.NoError(t, err)

	assert.Len(t, subs, 1)
	assert.Equal(t, myDid, subs[0])
}
