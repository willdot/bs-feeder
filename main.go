package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGTERM, syscall.SIGINT)

	feeder := NewFeedGenerator()

	server := NewServer(3000, feeder)
	go func() {
		<-signals

		_ = server.Stop(context.Background())
	}()

	server.Run()
}
