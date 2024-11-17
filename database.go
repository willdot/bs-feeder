package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"

	_ "github.com/glebarez/go-sqlite"
)

func NewDatabase(dbPath string) (*sql.DB, error) {
	err := createDbFile(dbPath)
	if err != nil {
		return nil, fmt.Errorf("create db file: %w", err)
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

	return db, nil
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

func createFeedTable(db *sql.DB) error {
	createFeedTableSQL := `CREATE TABLE IF NOT EXISTS feed (
		"id" integer NOT NULL PRIMARY KEY AUTOINCREMENT,
		"uri" TEXT,
		"userDID" TEXT
	  );`

	slog.Info("Create feed table...")
	statement, err := db.Prepare(createFeedTableSQL)
	if err != nil {
		return fmt.Errorf("prepare DB statement to create table: %w", err)
	}
	_, err = statement.Exec()
	if err != nil {
		return fmt.Errorf("exec sql statement to create table: %w", err)
	}
	slog.Info("feed table created")

	return nil
}

type feedItem struct {
	ID      int
	URI     string
	UserDID string
}

func addFeedItem(_ context.Context, db *sql.DB, feedItem feedItem) error {
	sql := `INSERT INTO feed (uri, userDID)
            VALUES (?, ?, ?);`
	_, err := db.Exec(sql, feedItem.URI, feedItem.UserDID)
	if err != nil {
		return fmt.Errorf("exec insert feed item: %w", err)
	}
	return nil
}

func getUsersFeedItems(db *sql.DB, usersDID string) ([]feedItem, error) {
	sql := "SELECT id, uri, userDID FROM feed WHERE userDID = ?"
	rows, err := db.Query(sql, usersDID)
	if err != nil {
		return nil, fmt.Errorf("run query to get users feed item: %w", err)
	}
	defer rows.Close()

	feedItems := make([]feedItem, 0)
	for rows.Next() {
		var feedItem feedItem
		if err := rows.Scan(&feedItem.ID, &feedItem.URI, &feedItem.UserDID); err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}
		feedItems = append(feedItems, feedItem)
	}

	return feedItems, nil
}
