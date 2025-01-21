package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	apibsky "github.com/bluesky-social/indigo/api/bsky"
	"github.com/bluesky-social/jetstream/pkg/models"
	"github.com/bugsnag/bugsnag-go/v2"
	"github.com/willdot/bskyfeedgen/store"
)

type HandlerStore interface {
	AddFeedPost(feedItem store.FeedPost) error
	GetBookmarksForPost(postURI string) ([]string, error)
}

type handler struct {
	store HandlerStore
}

func (h *handler) HandleEvent(ctx context.Context, event *models.Event) error {
	if event.Commit == nil {
		return nil
	}

	switch event.Commit.Operation {
	case models.CommitOperationCreate:
		return h.handleCreateEvent(ctx, event)
	default:
		return nil
	}
}

func (h *handler) handleCreateEvent(_ context.Context, event *models.Event) error {
	if event.Commit.Collection != "app.bsky.feed.post" {
		return nil
	}

	var post apibsky.FeedPost
	if err := json.Unmarshal(event.Commit.Record, &post); err != nil {
		// ignore this
		return nil
	}

	// we only care about posts that have parents which are replies
	if post.Reply == nil || post.Reply.Parent == nil || post.Reply.Parent.Uri == "" {
		return nil
	}

	subscribedPostURI := post.Reply.Parent.Uri

	// see if the post is a reply to a post we are subscribed to
	subscribedDids := h.getSubscribedDidsForPost(subscribedPostURI)
	if len(subscribedDids) == 0 {
		return nil
	}

	slog.Info("post is a reply to a post that users are subscribed to", "subscribed post URI", subscribedPostURI, "dids", subscribedDids, "RKey", event.Commit.RKey)

	createdAt, err := time.Parse(time.RFC3339, post.CreatedAt)
	if err != nil {
		slog.Error("parsing createdAt time from post", "error", err, "timestamp", post.CreatedAt)
		createdAt = time.Now().UTC()
	}

	replyPostURI := fmt.Sprintf("at://%s/app.bsky.feed.post/%s", event.Did, event.Commit.RKey)
	h.createFeedPostForSubscribedUsers(subscribedDids, replyPostURI, subscribedPostURI, createdAt.UnixMilli())
	return nil
}

func (h *handler) getSubscribedDidsForPost(postURI string) []string {
	// dids, err := h.store.GetSubscriptionsForPost(postURI)
	dids, err := h.store.GetBookmarksForPost(postURI)
	if err != nil {
		slog.Error("getting bookmarks for post", "error", err)
		_ = bugsnag.Notify(err)
	}

	return dids
}

func (h *handler) createFeedPostForSubscribedUsers(usersDids []string, replyPostURI, subscribedPostURI string, createdAt int64) {
	for _, did := range usersDids {
		feedItem := store.FeedPost{
			ReplyURI:          replyPostURI,
			UserDID:           did,
			SubscribedPostURI: subscribedPostURI,
			CreatedAt:         createdAt,
		}
		err := h.store.AddFeedPost(feedItem)
		if err != nil {
			slog.Error("add users feed item", "error", err, "did", did, "reply post URI", replyPostURI)
			_ = bugsnag.Notify(err)
			continue
		}
	}
}
