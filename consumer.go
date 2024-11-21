package main

import (
	"context"

	"fmt"
	"log/slog"
	"time"

	"github.com/bluesky-social/jetstream/pkg/client"
	"github.com/bluesky-social/jetstream/pkg/client/schedulers/sequential"
)

type consumer struct {
	cfg     *client.ClientConfig
	handler *handler
	logger  *slog.Logger
}

func NewConsumer(jsAddr string, logger *slog.Logger, handler *handler) *consumer {
	cfg := client.DefaultClientConfig()
	if jsAddr != "" {
		cfg.WebsocketURL = jsAddr
	}
	cfg.WantedCollections = []string{
		"app.bsky.feed.post",
	}
	cfg.WantedDids = []string{
		myDid,
	}

	return &consumer{
		cfg:     cfg,
		logger:  logger,
		handler: handler,
	}
}

func (c *consumer) Consume(ctx context.Context) error {
	scheduler := sequential.NewScheduler("jetstream_localdev", c.logger, c.handler.HandleEvent)
	defer scheduler.Shutdown()

	client, err := client.NewClient(c.cfg, c.logger, scheduler)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	cursor := time.Now().Add(1 * -time.Minute).UnixMicro()

	if err := client.ConnectAndRead(ctx, &cursor); err != nil {
		return fmt.Errorf("connect and read: %w", err)
	}

	slog.Info("stopping consume")
	return nil
}
