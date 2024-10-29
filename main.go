package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGTERM, syscall.SIGINT)

	feeder := NewFeedGenerator()

	feedDidBase := os.Getenv("FEED_DID_BASE")
	if feedDidBase == "" {
		slog.Error("FEED_DID_BASE not set")
		os.Exit(1)
	}
	appDid := os.Getenv("APP_DID")
	if appDid == "" {
		slog.Error("APP_DID not set")
		os.Exit(1)
	}

	server := NewServer(3000, feeder, appDid, feedDidBase)
	go func() {
		<-signals

		_ = server.Stop(context.Background())
	}()

	server.Run()
}
