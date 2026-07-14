package karu

import (
	"context"
	"fmt"
)

var migrations = map[string][]string{
	"postgres": {
		`CREATE TABLE IF NOT EXISTS posts (
			id         TEXT PRIMARY KEY,
			path       TEXT NOT NULL,
			parent_id  TEXT REFERENCES posts(id) ON DELETE CASCADE,
			author_id  TEXT NOT NULL,
			title      TEXT NOT NULL DEFAULT '',
			content    TEXT NOT NULL DEFAULT '',
			created_at BIGINT NOT NULL,
			updated_at BIGINT NOT NULL,
			is_locked  INTEGER NOT NULL DEFAULT 0,
			is_sticky  INTEGER NOT NULL DEFAULT 0,
			metadata   TEXT NOT NULL DEFAULT '{}'
		)`,
		`CREATE INDEX IF NOT EXISTS idx_posts_path ON posts(path)`,
		`CREATE INDEX IF NOT EXISTS idx_posts_parent ON posts(parent_id)`,
		`CREATE INDEX IF NOT EXISTS idx_posts_author ON posts(author_id)`,
		`CREATE INDEX IF NOT EXISTS idx_posts_created ON posts(created_at DESC)`,
		`CREATE TABLE IF NOT EXISTS preferences (
			user_id TEXT NOT NULL,
			key     TEXT NOT NULL,
			value   TEXT NOT NULL DEFAULT '',
			PRIMARY KEY (user_id, key)
		)`,
	},
	"sqlite": {
		`CREATE TABLE IF NOT EXISTS posts (
			id         TEXT PRIMARY KEY,
			path       TEXT NOT NULL,
			parent_id  TEXT REFERENCES posts(id) ON DELETE CASCADE,
			author_id  TEXT NOT NULL,
			title      TEXT NOT NULL DEFAULT '',
			content    TEXT NOT NULL DEFAULT '',
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL,
			is_locked  INTEGER NOT NULL DEFAULT 0,
			is_sticky  INTEGER NOT NULL DEFAULT 0,
			metadata   TEXT NOT NULL DEFAULT '{}'
		)`,
		`CREATE INDEX IF NOT EXISTS idx_posts_path ON posts(path)`,
		`CREATE INDEX IF NOT EXISTS idx_posts_parent ON posts(parent_id)`,
		`CREATE INDEX IF NOT EXISTS idx_posts_author ON posts(author_id)`,
		`CREATE INDEX IF NOT EXISTS idx_posts_created ON posts(created_at DESC)`,
		`CREATE TABLE IF NOT EXISTS preferences (
			user_id TEXT NOT NULL,
			key     TEXT NOT NULL,
			value   TEXT NOT NULL DEFAULT '',
			PRIMARY KEY (user_id, key)
		)`,
	},
}

func (k *Karu) Migrate(ctx context.Context) error {
	queries, ok := migrations[k.driver]
	if !ok {
		return fmt.Errorf("no migrations for driver: %s", k.driver)
	}
	for _, q := range queries {
		if _, err := k.db.ExecContext(ctx, q); err != nil {
			return fmt.Errorf("running migration: %w", err)
		}
	}
	return nil
}
