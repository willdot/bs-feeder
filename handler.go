package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	apibsky "github.com/bluesky-social/indigo/api/bsky"
	"github.com/bluesky-social/jetstream/pkg/models"
	"github.com/bugsnag/bugsnag-go/v2"
	"github.com/willdot/bskyfeedgen/store"
)

const (
	myDid = "did:plc:dadhhalkfcq3gucaq25hjqon"
)

type HandlerStore interface {
	AddFeedPost(feedItem store.FeedPost) error
	GetSubscriptionsForPost(postURI string) ([]string, error)
	AddSubscriptionForPost(subscribedPostURI, userDid, subscriptionPostRkey string) error
	GetSubscribedPostURI(userDID, subscriptionPostRkey string) (string, error)
	DeleteSubscriptionForUser(userDID, postURI string) error
	DeleteFeedPostsForSubscribedPostURIandUserDID(subscribedPostURI, userDID string) error
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
	case models.CommitOperationDelete:
		return h.handleDeleteEvent(ctx, event)
	default:
		return nil
	}
}

func (h *handler) handleCreateEvent(_ context.Context, event *models.Event) error {
	if event.Commit.Collection != "app.bsky.feed.post" {
		return nil
	}

	if event.Did == myDid {
		slog.Info("event from my did", "event", event.Commit.Record)
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

	// look for posts that are "subscribe" so that we can add the post URI to a list of posts we want to find replies for
	if strings.Contains(post.Text, "/subscribe") {
		// For now just look for me
		if event.Did != myDid {
			return nil
		}
		slog.Info("a post that's subscribing to another post. Adding to posts to look for", "subscribed post URI", subscribedPostURI)
		return h.addDidToSubscribedPost(subscribedPostURI, event.Did, event.Commit.RKey)
	}

	// see if the post is a reply to a post we are subscribed to
	subscribedDids := h.getSubscribedDidsForPost(subscribedPostURI)
	if len(subscribedDids) == 0 {
		return nil
	}

	slog.Info("post is a reply to a post that users are subscribed to", "subscribed post URI", subscribedPostURI, "dids", subscribedDids, "RKey", event.Commit.RKey)

	// TODO: parse from the event
	createdAt := time.Now().UTC().UnixMilli()

	replyPostURI := fmt.Sprintf("at://%s/app.bsky.feed.post/%s", event.Did, event.Commit.RKey)
	h.createFeedPostForSubscribedUsers(subscribedDids, replyPostURI, subscribedPostURI, createdAt)
	return nil
}

func (h *handler) handleDeleteEvent(_ context.Context, event *models.Event) error {
	if event.Commit.Collection != "app.bsky.feed.post" {
		return nil
	}

	// temp ignore everyone but me
	if event.Did != "did:plc:dadhhalkfcq3gucaq25hjqon" {
		return nil
	}
	slog.Info("delete event received", "did", event.Did, "rkey", event.Commit.RKey)
	subscribedPostURI, err := h.store.GetSubscribedPostURI(event.Did, event.Commit.RKey)
	if err != nil {
		slog.Error("get subscribed post URI", "error", err, "rkey", event.Commit.RKey, "user DID", event.Did)
		return fmt.Errorf("get subscribed post URI: %w", err)
	}

	//  delete from feeds for the subscribedPostURI and the users DID first. This is so that if this fails, it can be tried again and the
	// subscription will be still there
	err = h.store.DeleteFeedPostsForSubscribedPostURIandUserDID(subscribedPostURI, event.Did)
	if err != nil {
		slog.Error("delete feed items for subscribedPostURI and user", "error", err, "subscribedPostURI", subscribedPostURI, "user DID", event.Did)
		return fmt.Errorf("delete feed items for subscribedPostURI and user: %w", err)
	}

	// delete from subscriptions for the postURI and the users DID now that we have cleaned up the feeds
	err = h.store.DeleteSubscriptionForUser(event.Did, subscribedPostURI)
	if err != nil {
		slog.Error("delete subscription for user", "error", err, "subscribedPostURI", subscribedPostURI, "user DID", event.Did)
		return fmt.Errorf("delete subscription and user: %w", err)
	}

	return nil
}

func (h *handler) addDidToSubscribedPost(subscribedPostURI, userDid, subscriptionPostRkey string) error {
	err := h.store.AddSubscriptionForPost(subscribedPostURI, userDid, subscriptionPostRkey)
	if err != nil {
		return fmt.Errorf("add subscription for post: %w", err)
	}
	return nil
}

func (h *handler) getSubscribedDidsForPost(postURI string) []string {
	dids, err := h.store.GetSubscriptionsForPost(postURI)
	if err != nil {
		slog.Error("getting subscriptions for post", "error", err)
		bugsnag.Notify(err)
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
			bugsnag.Notify(err)
			continue
		}
	}
}
