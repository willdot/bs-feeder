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
		"createdAt" integer NOT NULL,
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

type FeedPost struct {
	ID                int
	ReplyURI          string
	UserDID           string
	SubscribedPostURI string
	CreatedAt         int64
}

func (s *Store) AddFeedPost(feedPost FeedPost) error {
	sql := `INSERT INTO feed (replyURI, userDID, subscribedPostURI, createdAt) VALUES (?, ?, ?, ?) ON CONFLICT(replyURI, userDID) DO NOTHING;`
	_, err := s.db.Exec(sql, feedPost.ReplyURI, feedPost.UserDID, feedPost.SubscribedPostURI, feedPost.CreatedAt)
	if err != nil {
		return fmt.Errorf("exec insert feed item: %w", err)
	}
	return nil
}

func (s *Store) GetUsersFeed(usersDID string, cursor int64, limit int) ([]FeedPost, error) {
	sql := `SELECT id, replyURI, userDID, subscribedPostURI, createdAt FROM feed
			WHERE userDID = ? AND createdAt < ?
			ORDER BY createdAt LIMIT ?;`
	rows, err := s.db.Query(sql, usersDID, cursor, limit)
	if err != nil {
		return nil, fmt.Errorf("run query to get users feed posts: %w", err)
	}
	defer rows.Close()

	feedPosts := make([]FeedPost, 0)
	for rows.Next() {
		var feedPost FeedPost
		if err := rows.Scan(&feedPost.ID, &feedPost.ReplyURI, &feedPost.UserDID, &feedPost.SubscribedPostURI, &feedPost.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}
		feedPosts = append(feedPosts, feedPost)
	}

	return feedPosts, nil
}

func (s *Store) DeleteFeedPostsForSubscribedPostURIandUserDID(subscribedPostURI, userDID string) error {
	sql := "DELETE FROM feed WHERE subscribedPostURI = ? AND userDID = ?;"
	statement, err := s.db.Prepare(sql)
	if err != nil {
		return fmt.Errorf("prepare delete feed posts: %w", err)
	}
	res, err := statement.Exec(subscribedPostURI, userDID)
	if err != nil {
		return fmt.Errorf("exec delete feed posts: %w", err)
	}

	n, _ := res.RowsAffected()

	slog.Info("delete feed posts result", "affected rows", n)
	return nil
}
