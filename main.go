package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path"
	"syscall"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/bugsnag/bugsnag-go/v2"
)

const (
	jsServerAddr = "wss://jetstream.atproto.tools/subscribe"
)

func main() {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGTERM, syscall.SIGINT)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	bugsnagAPIKey := os.Getenv("BUGSNAG_API_KEY")
	if bugsnagAPIKey != "" {
		bugsnag.Configure(bugsnag.Configuration{
			APIKey:       bugsnagAPIKey,
			ReleaseStage: "production",
			// The import paths for the Go packages containing your source files
			ProjectPackages: []string{"main", "github.com/willdot/bskyfeedgen"},
			// more configuration options
		})
	}

	dbMountPath := os.Getenv("RAILWAY_VOLUME_MOUNT_PATH")
	if dbMountPath == "" {
		bugsnag.Notify(fmt.Errorf("RAILWAY_VOLUME_MOUNT_PATH env not set"))
		return
	}
	dbFilename := path.Join(dbMountPath, "database.db")
	db, err := NewDatabase(dbFilename)
	if err != nil {
		slog.Error("create new database", "error", err)
		bugsnag.Notify(err)
		return
	}
	defer db.Close()

	feeder := NewFeedGenerator(db)

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
		slog.Info("enabling jetstream consume")
		go consumeLoop(ctx, jsServerAddr, feeder)
	}

	server := NewServer(443, feeder, feedHost, feedDidBase)
	go func() {
		<-signals
		cancel()
		_ = server.Stop(context.Background())
	}()

	server.Run()

	// give time for bugsnags to be sent
	time.Sleep(time.Second)
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
