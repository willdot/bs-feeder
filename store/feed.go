package store

import (
	"database/sql"
	"fmt"
	"log/slog"
)

func createFeedTable(db *sql.DB) error {
	createFeedTableSQL := `CREATE TABLE IF NOT EXISTS feed (
		"id" integer NOT NULL PRIMARY KEY AUTOINCREMENT,
		"replyURI" TEXT,
		"userDID" TEXT,
		"subscribedPostURI" TEXT,
		UNIQUE(replyURI, userDID)
	  );`

	slog.Info("Create feed table...")
	statement, err := db.Prepare(createFeedTableSQL)
	if err != nil {
		return fmt.Errorf("prepare DB statement to create feeds table: %w", err)
	}
	_, err = statement.Exec()
	if err != nil {
		return fmt.Errorf("exec sql statement to create feeds table: %w", err)
	}
	slog.Info("feed table created")

	return nil
}

type FeedItem struct {
	ID                int
	ReplyURI          string
	UserDID           string
	SubscribedPostURI string
}

func (s *Store) AddFeedItem(feedItem FeedItem) error {
	sql := `INSERT INTO feed (replyURI, userDID, subscribedPostURI) VALUES (?, ?, ?) ON CONFLICT(replyURI, userDID) DO NOTHING;`
	_, err := s.db.Exec(sql, feedItem.ReplyURI, feedItem.UserDID, feedItem.SubscribedPostURI)
	if err != nil {
		return fmt.Errorf("exec insert feed item: %w", err)
	}
	return nil
}

func (s *Store) GetUsersFeedItems(usersDID string) ([]FeedItem, error) {
	sql := "SELECT id, replyURI, userDID FROM feed WHERE userDID = ?;"
	rows, err := s.db.Query(sql, usersDID)
	if err != nil {
		return nil, fmt.Errorf("run query to get users feed item: %w", err)
	}
	defer rows.Close()

	feedItems := make([]FeedItem, 0)
	for rows.Next() {
		var feedItem FeedItem
		if err := rows.Scan(&feedItem.ID, &feedItem.ReplyURI, &feedItem.UserDID); err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}
		feedItems = append(feedItems, feedItem)
	}

	return feedItems, nil
}

func (s *Store) DeleteFeedItemsForSubscribedPostURIandUserDID(subscribedPostURI, userDID string) error {
	sql := "DELETE FROM feed WHERE subscribedPostURI = ? AND userDID = ?;"
	statement, err := s.db.Prepare(sql)
	if err != nil {
		return fmt.Errorf("prepare delete feed items: %w", err)
	}
	res, err := statement.Exec(subscribedPostURI, userDID)
	if err != nil {
		return fmt.Errorf("exec delete feed items: %w", err)
	}

	n, _ := res.RowsAffected()

	slog.Info("delete feed res", "affected rows", n)
	return nil
}
