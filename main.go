package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/avast/retry-go/v4"
)

const (
	jsServerAddr = "wss://jetstream.atproto.tools/subscribe"
)

func main() {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGTERM, syscall.SIGINT)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	feeder := NewFeedGenerator()

	feedDidBase := os.Getenv("FEED_DID_BASE")
	if feedDidBase == "" {
		slog.Error("FEED_DID_BASE not set")
		os.Exit(1)
	}
	feedHost := os.Getenv("FEED_HOST_NAME")
	if feedHost == "" {
		slog.Error("FEED_HOST_NAME not set")
		os.Exit(1)
	}

	enableJS := os.Getenv("ENABLE_JETSTREAM")
	if enableJS == "true" {
		go consumeLoop(ctx, jsServerAddr, feeder)
	}

	server := NewServer(443, feeder, feedHost, feedDidBase)
	go func() {
		<-signals
		cancel()
		_ = server.Stop(context.Background())
	}()

	server.Run()
}

func consumeLoop(ctx context.Context, jsServerAddr string, feeder *FeedGenerator) {
	consumer := NewConsumer(jsServerAddr)

	retry.Do(func() error {
		err := consumer.Consume(ctx, feeder, slog.Default())
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return nil
			}
			slog.Error("consume loop", "error", err)
			return err
		}
		return nil
	}, retry.Attempts(0))
}
