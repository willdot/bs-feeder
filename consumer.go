package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"sync"
	"time"

	apibsky "github.com/bluesky-social/indigo/api/bsky"
	"github.com/bluesky-social/jetstream/pkg/client"
	"github.com/bluesky-social/jetstream/pkg/client/schedulers/sequential"
	"github.com/bluesky-social/jetstream/pkg/models"
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
	cfg.WantedDids = []string{
		// "did:plc:dadhhalkfcq3gucaq25hjqon",
	}
	return &consumer{
		cfg: cfg,
	}
}

func (con *consumer) Consume(ctx context.Context, feedGen *FeedGenerator, logger *slog.Logger) error {
	h := &handler{
		seenSeqs:         make(map[int64]struct{}),
		feedGenerator:    feedGen,
		parentsToLookFor: make(map[string]struct{}),
	}

	scheduler := sequential.NewScheduler("jetstream_localdev", logger, h.HandleEvent)

	// TODO: logger
	c, err := client.NewClient(con.cfg, slog.Default(), scheduler)
	if err != nil {
		log.Fatalf("failed to create client: %v", err)
	}

	cursor := time.Now().Add(5 * -time.Minute).UnixMicro()

	if err := c.ConnectAndRead(ctx, &cursor); err != nil {
		return fmt.Errorf("connect and read: %w", err)
	}

	slog.Info("stopping consume")
	return nil
}

type handler struct {
	seenSeqs         map[int64]struct{}
	highwater        int64
	feedGenerator    *FeedGenerator
	mu               sync.Mutex
	parentsToLookFor map[string]struct{}
}

func (h *handler) HandleEvent(ctx context.Context, event *models.Event) error {
	// Unmarshal the record if there is one
	if event.Commit == nil {
		return nil
	}
	if event.Commit.Operation == models.CommitOperationCreate || event.Commit.Operation == models.CommitOperationUpdate {
		switch event.Commit.Collection {
		case "app.bsky.feed.post":
			var post apibsky.FeedPost
			if err := json.Unmarshal(event.Commit.Record, &post); err != nil {
				return fmt.Errorf("failed to unmarshal post: %w", err)
			}

			// we only care about posts that have parents which are replies
			if post.Reply == nil || post.Reply.Parent == nil || post.Reply.Parent.Uri == "" {
				return nil
			}

			h.mu.Lock()
			defer h.mu.Unlock()

			// look for posts where I've "subsribed" so that we can add the parent URI to a list of replies to that parent to look for
			if post.Text == "/subscribe" && event.Did == "did:plc:dadhhalkfcq3gucaq25hjqon" {
				slog.Info("it's a reply with a parent! Adding to parents to look for", "parent URI", post.Reply.Parent.Uri)
				h.parentsToLookFor[post.Reply.Parent.Uri] = struct{}{}
				return nil
			}

			// see if the post is a reply to a post we are subscribed to
			if _, ok := h.parentsToLookFor[post.Reply.Parent.Uri]; ok {
				slog.Info("post is a reply to a parent we are subscribed to", "parent URI", post.Reply.Parent.Uri, "did", event.Did, "RKey", event.Commit.RKey)
				h.feedGenerator.AddToFeedPosts(fmt.Sprintf("at://%s/app.bsky.feed.post/%s", event.Did, event.Commit.RKey))
			}
		}
	}
	return nil
}
