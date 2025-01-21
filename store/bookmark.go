package store

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
)

var ErrBookmarkAlreadyExists = errors.New("bookmark already exists")

func createBookmarksTable(db *sql.DB) error {
	createBooksmarksTableSQL := `CREATE TABLE IF NOT EXISTS bookmarks (
		"id" integer NOT NULL PRIMARY KEY AUTOINCREMENT,
		"postRKey" TEXT,
		"postURI" TEXT,
		"postATURI" TEXT,
		"authorDID" TEXT,
		"authorHandle" TEXT,
		"userDID" TEXT,
		"content" TEXT,
		UNIQUE(postRKey, userDID)
	  );`

	slog.Info("Create bookmarks table...")
	statement, err := db.Prepare(createBooksmarksTableSQL)
	if err != nil {
		return fmt.Errorf("prepare DB statement to create bookmarks table: %w", err)
	}
	_, err = statement.Exec()
	if err != nil {
		return fmt.Errorf("exec sql statement to create bookmarks table: %w", err)
	}
	slog.Info("bookmarks table created")

	return nil
}

type Bookmark struct {
	ID           int
	PostRKey     string
	PostURI      string
	PostATURI    string
	AuthorDID    string
	AuthorHandle string
	UserDID      string
	Content      string
}

func (s *Store) CreateBookmark(postRKey, postURI, postATURI, authorDID, authorHandle, userDID, content string) error {
	sql := `INSERT INTO bookmarks (postRKey, postURI,postATURI, authorDID, authorHandle, userDID, content) VALUES (?, ?, ?, ?, ?, ?, ?) ON CONFLICT(postRKey, userDID) DO NOTHING;`
	res, err := s.db.Exec(sql, postRKey, postURI, postATURI, authorDID, authorHandle, userDID, content)
	if err != nil {
		return fmt.Errorf("exec insert bookmark: %w", err)
	}

	if x, _ := res.RowsAffected(); x == 0 {
		return ErrBookmarkAlreadyExists
	}
	return nil
}

func (s *Store) GetBookmarksForUser(userDID string) ([]Bookmark, error) {
	sql := "SELECT id, postRKey, postURI, postATURI, authorDID, authorHandle,  userDID, content FROM bookmarks WHERE userDID = ?;"
	rows, err := s.db.Query(sql, userDID)
	if err != nil {
		return nil, fmt.Errorf("run query to get bookmarked posts for user: %w", err)
	}
	defer rows.Close()

	var results []Bookmark
	for rows.Next() {
		var bookmark Bookmark
		if err := rows.Scan(&bookmark.ID, &bookmark.PostRKey, &bookmark.PostURI, &bookmark.PostATURI, &bookmark.AuthorDID, &bookmark.AuthorHandle, &bookmark.UserDID, &bookmark.Content); err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}

		results = append(results, bookmark)
	}
	return results, nil
}

func (s *Store) DeleteBookmark(postRKey, userDID string) error {
	sql := "DELETE FROM bookmarks WHERE postRKey = ? AND userDID = ?;"
	_, err := s.db.Exec(sql, postRKey, userDID)
	if err != nil {
		return fmt.Errorf("exec delete bookmark by postRKey and userDID: %w", err)
	}
	return nil
}

func (s *Store) GetBookmarksForPost(postURI string) ([]string, error) {
	sql := "SELECT userDID FROM bookmarks WHERE postATURI = ?"
	rows, err := s.db.Query(sql, postURI)
	if err != nil {
		return nil, fmt.Errorf("run query to get bookmarks for post: %w", err)
	}
	defer rows.Close()

	dids := make([]string, 0)
	for rows.Next() {
		var bookmark Bookmark
		if err := rows.Scan(&bookmark.UserDID); err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}
		dids = append(dids, bookmark.UserDID)
	}

	return dids, nil
}

func (s *Store) GetBookmarkByRKeyForUser(rkey, userDID string) (*Bookmark, error) {
	sql := "SELECT id, postRKey, postURI, postATURI, authorDID, authorHandle,  userDID, content FROM bookmarks WHERE postRKey = ? AND userDID = ?;"
	rows, err := s.db.Query(sql, rkey, userDID)
	if err != nil {
		return nil, fmt.Errorf("run query to get bookmark by rkey and user: %w", err)
	}
	defer rows.Close()

	if rows.Next() {
		var bookmark Bookmark
		if err := rows.Scan(&bookmark.ID, &bookmark.PostRKey, &bookmark.PostURI, &bookmark.PostATURI, &bookmark.AuthorDID, &bookmark.AuthorHandle, &bookmark.UserDID, &bookmark.Content); err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}

		return &bookmark, nil
	}

	return nil, nil
}
