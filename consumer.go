package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
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
		"did:plc:dadhhalkfcq3gucaq25hjqon",
	}
	return &consumer{
		cfg: cfg,
	}
}

func (con *consumer) Consume(ctx context.Context, feedGen *FeedGenerator, logger *slog.Logger) error {
	h := &handler{
		seenSeqs:      make(map[int64]struct{}),
		feedGenerator: feedGen,
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
	seenSeqs      map[int64]struct{}
	highwater     int64
	feedGenerator *FeedGenerator
}

func (h *handler) HandleEvent(ctx context.Context, event *models.Event) error {
	// Unmarshal the record if there is one
	if event.Commit != nil && (event.Commit.Operation == models.CommitOperationCreate || event.Commit.Operation == models.CommitOperationUpdate) {
		switch event.Commit.Collection {
		case "app.bsky.feed.post":
			var post apibsky.FeedPost
			if err := json.Unmarshal(event.Commit.Record, &post); err != nil {
				return fmt.Errorf("failed to unmarshal post: %w", err)
			}

			// only look for posts where I've "subsribed"
			if post.Text != "/subscribe" {
				return nil
			}

			if post.Reply != nil && post.Reply.Parent != nil && post.Reply.Parent.Uri != "" {
				slog.Info("it's a reply with a parent! Adding to feeds", "parent URI", post.Reply.Parent.Uri)
				h.feedGenerator.AddToFeed(post.Reply.Parent.Uri)
			}
		}
	}

	return nil
}
