package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

func AutoMigrate(ctx context.Context, pool *pgxpool.Pool) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS teams (
			id SERIAL PRIMARY KEY,
			name TEXT NOT NULL UNIQUE
		);`,
		`CREATE TABLE IF NOT EXISTS users (
			user_id TEXT PRIMARY KEY,
			username TEXT NOT NULL,
			team_id INTEGER NOT NULL REFERENCES teams(id) ON DELETE RESTRICT,
			is_active BOOLEAN NOT NULL DEFAULT TRUE
		);`,
		`CREATE TABLE IF NOT EXISTS pull_requests (
			pr_id TEXT PRIMARY KEY,
			pr_name TEXT NOT NULL,
			author_id TEXT NOT NULL REFERENCES users(user_id) ON DELETE RESTRICT,
			status TEXT NOT NULL,
			created_at TIMESTAMPTZ,
			merged_at TIMESTAMPTZ
		);`,
		`CREATE TABLE IF NOT EXISTS pull_request_reviewers (
			pr_id TEXT NOT NULL REFERENCES pull_requests(pr_id) ON DELETE CASCADE,
			reviewer_id TEXT NOT NULL REFERENCES users(user_id) ON DELETE RESTRICT,
			PRIMARY KEY (pr_id, reviewer_id)
		);`,
	}

	for _, stmt := range stmts {
		if _, err := pool.Exec(ctx, stmt); err != nil {
			return err
		}
	}
	return nil
}
