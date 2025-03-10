package store

import (
	"database/sql"
	"fmt"
	"log/slog"
)

func createRepliesTable(db *sql.DB) error {
	createRepliesTableSQL := `CREATE TABLE IF NOT EXISTS replies (
		"id" integer NOT NULL PRIMARY KEY AUTOINCREMENT,
		"replyURI" TEXT,
		"userDID" TEXT,
		"subscribedPostURI" TEXT,
		"createdAt" integer NOT NULL,
		UNIQUE(replyURI, userDID)
	  );`

	slog.Info("Create replies table...")
	statement, err := db.Prepare(createRepliesTableSQL)
	if err != nil {
		return fmt.Errorf("prepare DB statement to create replies table: %w", err)
	}
	_, err = statement.Exec()
	if err != nil {
		return fmt.Errorf("exec sql statement to create replies table: %w", err)
	}
	slog.Info("replies table created")

	return nil
}

type ReplyPost struct {
	ID                int
	ReplyURI          string
	UserDID           string
	SubscribedPostURI string
	CreatedAt         int64
}

func (s *Store) AddRepliedPost(replyPost ReplyPost) error {
	sql := `INSERT INTO replies (replyURI, userDID, subscribedPostURI, createdAt) VALUES (?, ?, ?, ?) ON CONFLICT(replyURI, userDID) DO NOTHING;`
	_, err := s.db.Exec(sql, replyPost.ReplyURI, replyPost.UserDID, replyPost.SubscribedPostURI, replyPost.CreatedAt)
	if err != nil {
		return fmt.Errorf("exec insert replies post: %w", err)
	}
	return nil
}

func (s *Store) GetUsersReplies(usersDID string, cursor int64, limit int) ([]ReplyPost, error) {
	sql := `SELECT id, replyURI, userDID, subscribedPostURI, createdAt FROM replies
			WHERE userDID = ? AND createdAt < ?
			ORDER BY createdAt DESC LIMIT ?;`
	rows, err := s.db.Query(sql, usersDID, cursor, limit)
	if err != nil {
		return nil, fmt.Errorf("run query to get users replied posts: %w", err)
	}
	defer rows.Close()

	repliedPosts := make([]ReplyPost, 0)
	for rows.Next() {
		var replyPost ReplyPost
		if err := rows.Scan(&replyPost.ID, &replyPost.ReplyURI, &replyPost.UserDID, &replyPost.SubscribedPostURI, &replyPost.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}
		repliedPosts = append(repliedPosts, replyPost)
	}

	return repliedPosts, nil
}

func (s *Store) DeleteRepliedPostsForBookmarkedPostURIandUserDID(subscribedPostURI, userDID string) error {
	sql := "DELETE FROM replies WHERE subscribedPostURI = ? AND userDID = ?;"
	statement, err := s.db.Prepare(sql)
	if err != nil {
		return fmt.Errorf("prepare delete replies: %w", err)
	}
	res, err := statement.Exec(subscribedPostURI, userDID)
	if err != nil {
		return fmt.Errorf("exec delete replies: %w", err)
	}

	n, _ := res.RowsAffected()

	slog.Info("delete replies result", "affected rows", n)
	return nil
}
