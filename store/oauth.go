package store

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
)

var ErrOauthRequestAlreadyExists = errors.New("oauth request already exists")

func createOauthRequestsTable(db *sql.DB) error {
	createOauthRequestsTableSQL := `CREATE TABLE IF NOT EXISTS oauthrequests (
		"id" integer NOT NULL PRIMARY KEY AUTOINCREMENT,
		"authserverIss" TEXT,
		"state" TEXT,
		"did" TEXT,
		"pkceVerifier" TEXT,
		"dpopAuthserverNonce" TEXT,
		"dpopPrivateJwk" TEXT,
		UNIQUE(did,state)
	  );`

	slog.Info("Create oauthrequests table...")
	statement, err := db.Prepare(createOauthRequestsTableSQL)
	if err != nil {
		return fmt.Errorf("prepare DB statement to create oauthrequests table: %w", err)
	}
	_, err = statement.Exec()
	if err != nil {
		return fmt.Errorf("exec sql statement to create oauthrequests table: %w", err)
	}
	slog.Info("oauthrequests table created")

	return nil
}

type OauthRequest struct {
	ID                  uint
	AuthserverIss       string
	State               string
	Did                 string
	PkceVerifier        string
	DpopAuthserverNonce string
	DpopPrivateJwk      string
}

func (s *Store) CreateOauthRequest(request OauthRequest) error {
	sql := `INSERT INTO oauthrequests (authserverIss, state, did, pkceVerifier, dpopAuthServerNonce, dpopPrivateJwk) VALUES (?, ?, ?, ?, ?, ?) ON CONFLICT(did,state) DO NOTHING;`
	res, err := s.db.Exec(sql, request.AuthserverIss, request.State, request.Did, request.PkceVerifier, request.DpopAuthserverNonce, request.DpopPrivateJwk)
	if err != nil {
		return fmt.Errorf("exec insert oauth request: %w", err)
	}

	if x, _ := res.RowsAffected(); x == 0 {
		return ErrOauthRequestAlreadyExists
	}
	return nil
}

func (s *Store) GetOauthRequest(state string) (OauthRequest, error) {
	var oauthRequest OauthRequest
	sql := "SELECT authserverIss, state, did, pkceVerifier, dpopAuthServerNonce, dpopPrivateJwk FROM oauthrequests WHERE state = ?;"
	rows, err := s.db.Query(sql, state)
	if err != nil {
		return oauthRequest, fmt.Errorf("run query to get oauth request: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		if err := rows.Scan(&oauthRequest.AuthserverIss, &oauthRequest.State, &oauthRequest.Did, &oauthRequest.PkceVerifier, &oauthRequest.DpopAuthserverNonce, &oauthRequest.DpopPrivateJwk); err != nil {
			return oauthRequest, fmt.Errorf("scan row: %w", err)
		}

		return oauthRequest, nil
	}
	return oauthRequest, fmt.Errorf("not found")
}

func (s *Store) DeleteOauthRequest(state string) error {
	sql := "DELETE FROM oauthrequests WHERE state = ?;"
	_, err := s.db.Exec(sql, state)
	if err != nil {
		return fmt.Errorf("exec delete oauth request: %w", err)
	}
	return nil
}
