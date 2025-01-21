package main

import (
	"context"
	"errors"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"path"
	"syscall"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/bugsnag/bugsnag-go/v2"
	"github.com/joho/godotenv"
	"github.com/willdot/bskyfeedgen/store"
)

const (
	defaultServerAddr = "wss://jetstream.atproto.tools/subscribe"
)

func main() {
	configureLogger()

	err := godotenv.Load()
	if err != nil {
		if !os.IsNotExist(err) {
			log.Fatal("Error loading .env file")
		}
	}

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGTERM, syscall.SIGINT)

	enableJS := os.Getenv("ENABLE_JETSTREAM")
	bugsnagAPIKey := os.Getenv("BUGSNAG_API_KEY")

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
	dbMountPath := os.Getenv("RAILWAY_VOLUME_MOUNT_PATH")
	if dbMountPath == "" {
		slog.Error("RAILWAY_VOLUME_MOUNT_PATH env not set")
		os.Exit(1)
	}

	if bugsnagAPIKey != "" {
		bugsnag.Configure(bugsnag.Configuration{
			APIKey:       bugsnagAPIKey,
			ReleaseStage: "production",
			// The import paths for the Go packages containing your source files
			ProjectPackages: []string{"main", "github.com/willdot/bskyfeedgen"},
			// more configuration options
			AutoCaptureSessions: false,
		})
	}

	dbFilename := path.Join(dbMountPath, "database.db")
	store, err := store.New(dbFilename)
	if err != nil {
		slog.Error("create new store", "error", err)
		_ = bugsnag.Notify(err)
		return
	}
	defer store.Close()

	feeder := NewFeedGenerator(store)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if enableJS == "true" {
		slog.Info("enabling jetstream consume")
		go consumeLoop(ctx, store)
	}

	server := NewServer(443, feeder, feedHost, feedDidBase, store)
	go func() {
		<-signals
		cancel()
		_ = server.Stop(context.Background())
	}()

	server.Run()

	// give time for bugsnags to be sent
	time.Sleep(time.Second)
}

func consumeLoop(ctx context.Context, store *store.Store) {
	handler := handler{
		store: store,
	}

	jsServerAddr := os.Getenv("JS_SERVER_ADDR")
	if jsServerAddr == "" {
		jsServerAddr = defaultServerAddr
	}

	consumer := NewConsumer(jsServerAddr, slog.Default(), &handler)

	_ = retry.Do(func() error {
		err := consumer.Consume(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return nil
			}
			slog.Error("consume loop", "error", err)
			return err
		}
		return nil
	}, retry.Attempts(0)) // retry indefinitly until context canceled

	slog.Warn("exiting consume loop")
}
