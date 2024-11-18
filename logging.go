package main

import (
	"log/slog"
	"os"
)

func configureLogger() {
	minimumLevel := slog.LevelInfo

	logger := createLogger(minimumLevel)
	slog.SetDefault(logger)
}

func createLogger(minimumLevel slog.Leveler) *slog.Logger {
	commonFields := []slog.Attr{} // TODO: add common fields we may want
	changeNameFields := func(groups []string, a slog.Attr) slog.Attr {
		if a.Key == slog.LevelKey {
			a.Key = "level"
			return a
		}
		return a
	}
	h := slog.NewJSONHandler(
		os.Stdout,
		&slog.HandlerOptions{
			Level:       minimumLevel,
			ReplaceAttr: changeNameFields,
		},
	).WithAttrs(
		commonFields,
	)

	l := slog.New(h)
	return l
}
