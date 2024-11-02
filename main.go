package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
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
		consumer := NewConsumer(jsServerAddr)
		go func() {
			err := consumer.Consume(ctx, slog.Default())
			if err != nil {
				slog.Error("consume", "error", err)
			}
		}()
	}

	server := NewServer(443, feeder, feedHost, feedDidBase)
	go func() {
		<-signals
		cancel()
		_ = server.Stop(context.Background())
	}()

	server.Run()
}
