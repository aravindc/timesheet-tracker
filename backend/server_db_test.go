package main

import (
	"database/sql"
	"errors"
	"os"
	"testing"

	"github.com/lib/pq"
	"go.uber.org/zap"
)

// openTestDB connects to TEST_DATABASE_URL, a throwaway database whose
// schema this test drops. The test is skipped when it is not set.
func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	db, err := sql.Open("postgres", url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if _, err := db.Exec(`DROP SCHEMA public CASCADE; CREATE SCHEMA public`); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestProjectNamesUniquePerUser(t *testing.T) {
	tests := []struct {
		name  string
		setup string
	}{
		{"fresh database", ""},
		// Schema as created before per-user projects: name globally unique.
		{"upgraded database", `
			CREATE TABLE users (id SERIAL PRIMARY KEY, username VARCHAR NOT NULL UNIQUE, password_hash VARCHAR NOT NULL, created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP);
			CREATE TABLE projects (id SERIAL PRIMARY KEY, name VARCHAR NOT NULL UNIQUE, description TEXT, created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP);`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := openTestDB(t)
			if tt.setup != "" {
				if _, err := db.Exec(tt.setup); err != nil {
					t.Fatal(err)
				}
			}
			s := &Server{db: db, logger: zap.NewNop()}
			if err := s.initDB(); err != nil {
				t.Fatalf("initDB: %v", err)
			}
			// initDB runs on every startup, so it must be idempotent.
			if err := s.initDB(); err != nil {
				t.Fatalf("second initDB: %v", err)
			}

			var alice, bob int
			db.QueryRow(`INSERT INTO users (username) VALUES ('alice') RETURNING id`).Scan(&alice)
			db.QueryRow(`INSERT INTO users (username) VALUES ('bob') RETURNING id`).Scan(&bob)

			insert := func(userID int) error {
				_, err := db.Exec(`INSERT INTO projects (name, user_id) VALUES ('Internal', $1)`, userID)
				return err
			}
			if err := insert(alice); err != nil {
				t.Fatalf("alice's project: %v", err)
			}
			if err := insert(bob); err != nil {
				t.Fatalf("bob should be able to reuse alice's project name: %v", err)
			}
			var pqErr *pq.Error
			if err := insert(alice); !errors.As(err, &pqErr) || pqErr.Code != "23505" {
				t.Fatalf("duplicate name for the same user: got %v, want unique violation", err)
			}
		})
	}
}
