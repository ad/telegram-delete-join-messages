package data

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

func InitSqliteDB(path string) (*sql.DB, error) {
	// fmt.Println(path)

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(1)

	errInitVotes := initSqliteVotes(db)
	if errInitVotes != nil {
		return nil, errInitVotes
	}

	return db, nil
}

func InitPostgresDB(dsn string) (*sql.DB, error) {
	fmt.Println("InitPostgresDB")

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}

	errInitVotes := initPostgresVotes(db)
	if errInitVotes != nil {
		return nil, errInitVotes
	}

	return db, nil
}

func initSqliteVotes(db *sql.DB) error {
	_, err := db.Exec(`
CREATE TABLE IF NOT EXISTS "votes"  (
  "id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
  "user_id" integer NOT NULL,
  "group_id" integer NOT NULL DEFAULT 0,
  "vote" integer NOT NULL DEFAULT 0,
  "state" integer NOT NULL DEFAULT 0,
  "timestamp_created" DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT "votes_uniq" UNIQUE ("user_id" ASC, "group_id" ASC)
);
`)
	return err
}

func initPostgresVotes(db *sql.DB) error {
	_, err := db.Exec(`
CREATE TABLE IF NOT EXISTS votes (
  id SERIAL PRIMARY KEY,
  user_id integer NOT NULL,
  group_id integer NOT NULL DEFAULT 0,
  vote integer NOT NULL DEFAULT 0,
  state integer NOT NULL DEFAULT 0,
  timestamp_created TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT votes_uniq UNIQUE (user_id, group_id)
);
`)

	return err
}
