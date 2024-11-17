package main

import (
	"database/sql"
	"fmt"
	"log"
	"log/slog"
	"os"

	"github.com/bugsnag/bugsnag-go/v2"
	_ "github.com/glebarez/go-sqlite"
)

func db() {
	os.Remove("sqlite-database.db") // I delete the file to avoid duplicated records.
	// SQLite is a file based database.

	log.Println("Creating sqlite-database.db...")
	file, err := os.Create("sqlite-database.db") // Create SQLite file
	if err != nil {
		bugsnag.Notify(fmt.Errorf("create db: %w", err))
		return
	}
	file.Close()
	log.Println("sqlite-database.db created")

	sqliteDatabase, _ := sql.Open("sqlite", "./sqlite-database.db") // Open the created SQLite File
	defer sqliteDatabase.Close()

	createTable(sqliteDatabase)
	read(sqliteDatabase)
}

func createTable(db *sql.DB) {
	createFeedTableSQL := `CREATE TABLE feed (
		"id" integer NOT NULL PRIMARY KEY AUTOINCREMENT,
		"uri" TEXT,
		"userDID" TEXT
	  );`

	slog.Info("Create feed table...")
	statement, err := db.Prepare(createFeedTableSQL)
	if err != nil {
		bugsnag.Notify(fmt.Errorf("prepare DB statement: %w", err))
		return
	}
	_, err = statement.Exec()
	if err != nil {
		bugsnag.Notify(fmt.Errorf("exec sql statement: %w", err))
		return
	}
	slog.Info("feed table created")

	_, err = db.Exec("INSERT INTO feed(uri, userDID) VALUES(?,?)", "hello", "world")
	if err != nil {
		bugsnag.Notify(fmt.Errorf("insert into table: %w", err))
		return
	}
}

type feedItem struct {
	ID      int
	URI     string
	UserDID string
}

func read(db *sql.DB) {
	rows, err := db.Query("SELECT id, uri, userDID FROM feed")
	if err != nil {
		bugsnag.Notify(fmt.Errorf("db query: %w", err))
		return
	}
	defer rows.Close() // Ensure rows are closed after processing

	feedItems := make([]feedItem, 0) // Slice to store todos
	for rows.Next() {
		var feedItem feedItem
		if err := rows.Scan(&feedItem.ID, &feedItem.URI, &feedItem.UserDID); err != nil {
			bugsnag.Notify(fmt.Errorf("db scan: %w", err))
			return
		}
		feedItems = append(feedItems, feedItem)
	}

	slog.Info("feed items read", "values", feedItems)
}
