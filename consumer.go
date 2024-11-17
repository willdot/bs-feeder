package main

import (
	"context"
	"database/sql"
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
		db:            feedGen.db,
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
	db            *sql.DB
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

	parentURI := post.Reply.Parent.Uri

	// look for posts where I've "subsribed" so that we can add the parent URI to a list of replies to that parent to look for
	if strings.Contains(post.Text, "/subscribe") && event.Did == "did:plc:dadhhalkfcq3gucaq25hjqon" {
		slog.Info("a post that's subscribing to a parent. Adding to parents to look for", "parent URI", parentURI)
		return h.addDidToSubscribedParent(parentURI, event.Did, event.Commit.RKey)
	}

	// see if the post is a reply to a post we are subscribed to
	subscribedDids := h.getSubscribedDidsForParent(parentURI)
	if len(subscribedDids) == 0 {
		return nil
	}

	slog.Info("post is a reply to a parent that users are subscribed to", "parent URI", parentURI, "dids", subscribedDids, "RKey", event.Commit.RKey)

	h.feedGenerator.AddToFeedPosts(subscribedDids, parentURI, fmt.Sprintf("at://%s/app.bsky.feed.post/%s", event.Did, event.Commit.RKey))
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

	parentURI, err := getSubscribingPostParentURI(h.db, event.Did, event.Commit.RKey)
	if err != nil {
		slog.Error("get subscribing post parent URI", "error", err, "rkey", event.Commit.RKey, "user DID", event.Did)
		return fmt.Errorf("get subscribing post parent URI: %w", err)
	}

	slog.Info("delete parent URI", "parent URI", parentURI, "rkey", event.Commit.RKey)

	//  delete from feeds for the parentURI and the users DID first. This is so that if this fails, it can be tried again and the
	// subscription will be still there
	err = deleteFeedItemsForParentURIandUserDID(h.db, parentURI, event.Did)
	if err != nil {
		slog.Error("delete feed items for parentURI and user", "error", err, "parentURI", parentURI, "user DID", event.Did)
		return fmt.Errorf("delete feed items for parentURI and user: %w", err)
	}

	// delete from subscriptions for the parentURI and the users DID now that we have cleaned up the feeds
	err = deleteSubscriptionForUser(h.db, event.Did, parentURI)
	if err != nil {
		slog.Error("delete subscription for user", "error", err, "parentURI", parentURI, "user DID", event.Did)
		return fmt.Errorf("delete subscription and user: %w", err)
	}

	return nil
}

func (h *handler) addDidToSubscribedParent(parentURI, userDid, rkey string) error {
	err := addSubscriptionForParent(h.db, parentURI, userDid, rkey)
	if err != nil {
		return fmt.Errorf("add subscription for parent: %w", err)
	}
	return nil
}

func (h *handler) getSubscribedDidsForParent(parentURI string) []string {
	dids, err := getSubscriptionsForParent(h.db, parentURI)
	if err != nil {
		slog.Error("getting subscriptions for parent", "error", err)
		bugsnag.Notify(err)
	}

	return dids
}
