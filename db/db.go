package db

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

func Connect(databaseURL string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(context.Background()); err != nil {
		return nil, err
	}
	return pool, nil
}

func Migrate(pool *pgxpool.Pool) error {
	_, err := pool.Exec(context.Background(), `
		CREATE EXTENSION IF NOT EXISTS pgcrypto;

		CREATE TABLE IF NOT EXISTS character_sets (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name TEXT NOT NULL,
			description TEXT DEFAULT '',
			global_style TEXT DEFAULT '',
			created_at TIMESTAMPTZ DEFAULT NOW()
		);

		CREATE TABLE IF NOT EXISTS character_set_images (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			character_set_id UUID NOT NULL REFERENCES character_sets(id) ON DELETE CASCADE,
			storage_key TEXT NOT NULL,
			created_at TIMESTAMPTZ DEFAULT NOW()
		);

		CREATE TABLE IF NOT EXISTS batch_jobs (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			character_set_id UUID NOT NULL REFERENCES character_sets(id) ON DELETE CASCADE,
			title TEXT DEFAULT '',
			global_style TEXT DEFAULT '',
			status TEXT NOT NULL DEFAULT 'queued',
			width INT NOT NULL DEFAULT 1024,
			height INT NOT NULL DEFAULT 1024,
			total_count INT NOT NULL DEFAULT 0,
			completed_count INT NOT NULL DEFAULT 0,
			failed_count INT NOT NULL DEFAULT 0,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW()
		);

		CREATE TABLE IF NOT EXISTS batch_items (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			batch_job_id UUID NOT NULL REFERENCES batch_jobs(id) ON DELETE CASCADE,
			prompt_index INT NOT NULL,
			prompt_text TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'queued',
			image_key TEXT,
			error TEXT DEFAULT '',
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW()
		);
	`)
	return err
}
