package store

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"

	_ "github.com/glebarez/go-sqlite"
)

type Store struct {
	db *sql.DB
}

func New(dbPath string) (*Store, error) {
	if dbPath != ":memory:" {
		err := createDbFile(dbPath)
		if err != nil {
			return nil, fmt.Errorf("create db file: %w", err)
		}
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	err = db.Ping()
	if err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}

	err = createFeedTable(db)
	if err != nil {
		return nil, fmt.Errorf("creating feed table: %w", err)
	}

	err = createSubscriptionsTable(db)
	if err != nil {
		return nil, fmt.Errorf("creating subscription table: %w", err)
	}

	err = createBookmarksTable(db)
	if err != nil {
		return nil, fmt.Errorf("creating bookmarks table: %w", err)
	}

	return &Store{db: db}, nil
}

func (s *Store) Close() {
	err := s.db.Close()
	if err != nil {
		slog.Error("failed to close db", "error", err)
	}
}

func createDbFile(dbFilename string) error {
	if _, err := os.Stat(dbFilename); !errors.Is(err, os.ErrNotExist) {
		return nil
	}

	f, err := os.Create(dbFilename)
	if err != nil {
		return fmt.Errorf("create db file : %w", err)
	}
	f.Close()
	return nil
}
