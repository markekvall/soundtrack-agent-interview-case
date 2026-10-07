// Package store opens the SQLite database and keeps its schema up to date.
package store

import (
	"database/sql"

	_ "modernc.org/sqlite"
)

const schema = `
CREATE TABLE IF NOT EXISTS usage_reports (
	id         INTEGER PRIMARY KEY AUTOINCREMENT,
	user_id    TEXT NOT NULL,
	track_id   TEXT NOT NULL,
	platform   TEXT NOT NULL,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS usage_reports_user_id ON usage_reports (user_id);
`

func Open(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}
