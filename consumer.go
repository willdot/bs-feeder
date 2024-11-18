package main

import (
	"context"

	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	apibsky "github.com/bluesky-social/indigo/api/bsky"
	"github.com/bluesky-social/jetstream/pkg/client"
	"github.com/bluesky-social/jetstream/pkg/client/schedulers/sequential"
	"github.com/bluesky-social/jetstream/pkg/models"
	"github.com/bugsnag/bugsnag-go/v2"
	"github.com/willdot/bskyfeedgen/store"
)

type consumer struct {
	cfg *client.ClientConfig
}

func NewConsumer(jsAddr string) *consumer {
	cfg := client.DefaultClientConfig()
	if jsAddr != "" {
		cfg.WebsocketURL = jsAddr
	}
	cfg.WantedCollections = []string{
		"app.bsky.feed.post",
	}
	cfg.WantedDids = []string{}
	return &consumer{
		cfg: cfg,
	}
}

func (con *consumer) Consume(ctx context.Context, feedGen *FeedGenerator, logger *slog.Logger) error {
	h := &handler{
		seenSeqs:      make(map[int64]struct{}),
		feedGenerator: feedGen,
		store:         *feedGen.store,
	}

	scheduler := sequential.NewScheduler("jetstream_localdev", logger, h.HandleEvent)
	defer scheduler.Shutdown()

	// TODO: logger
	c, err := client.NewClient(con.cfg, slog.Default(), scheduler)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	cursor := time.Now().Add(1 * -time.Minute).UnixMicro()

	if err := c.ConnectAndRead(ctx, &cursor); err != nil {
		return fmt.Errorf("connect and read: %w", err)
	}

	slog.Info("stopping consume")
	return nil
}

type handler struct {
	seenSeqs      map[int64]struct{}
	highwater     int64
	feedGenerator *FeedGenerator
	store         store.Store
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
	if strings.Contains(post.Text, "/subscribe") && event.Did == "did:plc:dadhhalkfcq3gucaq25hjqon" {
		slog.Info("a post that's subscribing to another post. Adding to posts to look for", "subscribed post URI", subscribedPostURI)
		return h.addDidToSubscribedPost(subscribedPostURI, event.Did, event.Commit.RKey)
	}

	// see if the post is a reply to a post we are subscribed to
	subscribedDids := h.getSubscribedDidsForPost(subscribedPostURI)
	if len(subscribedDids) == 0 {
		return nil
	}

	slog.Info("post is a reply to a post that users are subscribed to", "subscribed post URI", subscribedPostURI, "dids", subscribedDids, "RKey", event.Commit.RKey)

	replyPostURI := fmt.Sprintf("at://%s/app.bsky.feed.post/%s", event.Did, event.Commit.RKey)
	h.feedGenerator.AddToFeedPosts(subscribedDids, subscribedPostURI, replyPostURI)
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
	err = h.store.DeleteFeedItemsForSubscribedPostURIandUserDID(subscribedPostURI, event.Did)
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

func (h *handler) addDidToSubscribedPost(subscribedPostURI, userDid, rkey string) error {
	err := h.store.AddSubscriptionForPost(subscribedPostURI, userDid, rkey)
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
