package store

import (
	"database/sql"
	"fmt"
	"log/slog"
)

func createSubscriptionsTable(db *sql.DB) error {
	createSubscriptionsTableSQL := `CREATE TABLE IF NOT EXISTS subscriptions (
		"id" integer NOT NULL PRIMARY KEY AUTOINCREMENT,
		"subscribedPostURI" TEXT,
		"userDID" TEXT,
		"subscriptionPostRkey" TEXT,
		UNIQUE(subscribedPostURI, userDID)
	  );`

	slog.Info("Create subscriptions table...")
	statement, err := db.Prepare(createSubscriptionsTableSQL)
	if err != nil {
		return fmt.Errorf("prepare DB statement to create subscriptions table: %w", err)
	}
	_, err = statement.Exec()
	if err != nil {
		return fmt.Errorf("exec sql statement to create subscriptions table: %w", err)
	}
	slog.Info("subscriptions table created")

	return nil
}

type Subscription struct {
	ID                   int
	SubscribedPostURI    string
	UserDID              string
	SubscriptionPostRkey string
}

func (s *Store) GetSubscriptionsForPost(postURI string) ([]string, error) {
	sql := "SELECT userDID FROM subscriptions WHERE subscribedPostURI = ?"
	rows, err := s.db.Query(sql, postURI)
	if err != nil {
		return nil, fmt.Errorf("run query to get subscriptions: %w", err)
	}
	defer rows.Close()

	dids := make([]string, 0)
	for rows.Next() {
		var subscription Subscription
		if err := rows.Scan(&subscription.UserDID); err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}
		dids = append(dids, subscription.UserDID)
	}

	return dids, nil
}

func (s *Store) AddSubscriptionForPost(subscribedPostURI, userDid, subscriptionRkey string) error {
	sql := `INSERT INTO subscriptions (subscribedPostURI, userDID, subscriptionPostRkey) VALUES (?, ?, ?) ON CONFLICT(subscribedPostURI, userDID) DO NOTHING;`
	_, err := s.db.Exec(sql, subscribedPostURI, userDid, subscriptionRkey)
	if err != nil {
		return fmt.Errorf("exec insert subscrptions: %w", err)
	}
	return nil
}

func (s *Store) GetSubscribedPostURI(userDID, rkey string) (string, error) {
	sql := "SELECT id, subscribedPostURI FROM subscriptions WHERE subscriptionPostRkey = ? AND userDID = ?;"
	rows, err := s.db.Query(sql, rkey, userDID)
	if err != nil {
		return "", fmt.Errorf("run query to get subscribed post URI: %w", err)
	}
	defer rows.Close()

	subscribedPostURI := ""
	for rows.Next() {
		var subscription Subscription
		if err := rows.Scan(&subscription.ID, &subscription.SubscribedPostURI); err != nil {
			return "", fmt.Errorf("scan row: %w", err)
		}

		subscribedPostURI = subscription.SubscribedPostURI
		break
	}
	return subscribedPostURI, nil
}

func (s *Store) DeleteSubscriptionForUser(userDID, postURI string) error {
	sql := "DELETE FROM subscriptions WHERE subscribedPostURI = ? AND userDID = ?;"
	_, err := s.db.Exec(sql, postURI, userDID)
	if err != nil {
		return fmt.Errorf("exec delete subscription for user: %w", err)
	}
	return nil
}
