package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"strings"
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
	cfg.WantedDids = []string{}
	return &consumer{
		cfg: cfg,
	}
}

func (con *consumer) Consume(ctx context.Context, feedGen *FeedGenerator, logger *slog.Logger) error {
	h := &handler{
		seenSeqs:         make(map[int64]struct{}),
		feedGenerator:    feedGen,
		parentsToLookFor: make(map[string]map[string]struct{}),
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
	parentsToLookFor map[string]map[string]struct{}
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

			parentURI := post.Reply.Parent.Uri

			// look for posts where I've "subsribed" so that we can add the parent URI to a list of replies to that parent to look for
			if strings.Contains(post.Text, "/subscribe") && event.Did == "did:plc:dadhhalkfcq3gucaq25hjqon" {
				slog.Info("a post that's subscribing to a parent. Adding to parents to look for", "parent URI", parentURI)
				h.addDidToSubscribedParent(parentURI, event.Did)
				return nil
			}

			// see if the post is a reply to a post we are subscribed to
			subscribedDids := h.getSubscribedDidsForParent(parentURI)
			if len(subscribedDids) == 0 {
				return nil
			}

			slog.Info("post is a reply to a parent that users are subscribed to", "parent URI", parentURI, "dids", subscribedDids, "RKey", event.Commit.RKey)

			h.feedGenerator.AddToFeedPosts(subscribedDids, fmt.Sprintf("at://%s/app.bsky.feed.post/%s", event.Did, event.Commit.RKey))
		}
	}
	return nil
}

func (h *handler) addDidToSubscribedParent(parentURI, did string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	subscribedDids, ok := h.parentsToLookFor[parentURI]
	if !ok {
		h.parentsToLookFor[parentURI] = map[string]struct{}{
			did: struct{}{},
		}
		return
	}

	subscribedDids[did] = struct{}{}
	h.parentsToLookFor[parentURI] = subscribedDids
}

func (h *handler) getSubscribedDidsForParent(parentURI string) []string {
	h.mu.Lock()
	defer h.mu.Unlock()

	subscribedDids, ok := h.parentsToLookFor[parentURI]
	if !ok {
		return nil
	}

	dids := make([]string, 0, len(subscribedDids))
	for did := range subscribedDids {
		dids = append(dids, did)
	}

	return dids
}
