package services

import (
	"context"

	"synthframe-api/models"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) CreateCharacterSet(ctx context.Context, input models.CreateCharacterSetInput) (models.CharacterSet, error) {
	var set models.CharacterSet
	err := r.db.QueryRow(ctx, `
		INSERT INTO character_sets (name, description, global_style)
		VALUES ($1, $2, $3)
		RETURNING id::text, name, description, global_style, created_at
	`, input.Name, input.Description, input.GlobalStyle).Scan(
		&set.ID,
		&set.Name,
		&set.Description,
		&set.GlobalStyle,
		&set.CreatedAt,
	)
	return set, err
}

func (r *Repository) AddCharacterReference(ctx context.Context, characterSetID, storageKey string) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO character_set_images (character_set_id, storage_key)
		VALUES ($1, $2)
	`, characterSetID, storageKey)
	return err
}

func (r *Repository) ListCharacterSets(ctx context.Context) ([]models.CharacterSet, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id::text, name, description, global_style, created_at
		FROM character_sets
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	sets := []models.CharacterSet{}
	for rows.Next() {
		var set models.CharacterSet
		if err := rows.Scan(&set.ID, &set.Name, &set.Description, &set.GlobalStyle, &set.CreatedAt); err != nil {
			return nil, err
		}
		refs, err := r.listCharacterReferences(ctx, set.ID)
		if err != nil {
			return nil, err
		}
		set.References = refs
		sets = append(sets, set)
	}
	return sets, rows.Err()
}

func (r *Repository) GetCharacterSet(ctx context.Context, id string) (models.CharacterSet, error) {
	var set models.CharacterSet
	err := r.db.QueryRow(ctx, `
		SELECT id::text, name, description, global_style, created_at
		FROM character_sets
		WHERE id = $1
	`, id).Scan(&set.ID, &set.Name, &set.Description, &set.GlobalStyle, &set.CreatedAt)
	if err != nil {
		return set, err
	}
	refs, err := r.listCharacterReferences(ctx, set.ID)
	if err != nil {
		return set, err
	}
	set.References = refs
	return set, nil
}

func (r *Repository) listCharacterReferences(ctx context.Context, id string) ([]models.CharacterReference, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id::text, storage_key, created_at
		FROM character_set_images
		WHERE character_set_id = $1
		ORDER BY created_at ASC
	`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	refs := []models.CharacterReference{}
	for rows.Next() {
		var ref models.CharacterReference
		if err := rows.Scan(&ref.ID, &ref.StorageKey, &ref.CreatedAt); err != nil {
			return nil, err
		}
		refs = append(refs, ref)
	}
	return refs, rows.Err()
}

func (r *Repository) CreateBatch(ctx context.Context, input models.CreateBatchInput) (models.BatchJob, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return models.BatchJob{}, err
	}
	defer tx.Rollback(ctx)

	batch := models.BatchJob{}
	err = tx.QueryRow(ctx, `
		INSERT INTO batch_jobs (character_set_id, title, global_style, status, width, height, total_count, completed_count, failed_count)
		VALUES ($1, $2, $3, 'queued', $4, $5, $6, 0, 0)
		RETURNING id::text, character_set_id::text, title, global_style, status, width, height, total_count, completed_count, failed_count, created_at, updated_at
	`, input.CharacterSetID, input.Title, input.GlobalStyle, input.Width, input.Height, len(input.Prompts)).Scan(
		&batch.ID,
		&batch.CharacterSetID,
		&batch.Title,
		&batch.GlobalStyle,
		&batch.Status,
		&batch.Width,
		&batch.Height,
		&batch.TotalCount,
		&batch.CompletedCount,
		&batch.FailedCount,
		&batch.CreatedAt,
		&batch.UpdatedAt,
	)
	if err != nil {
		return models.BatchJob{}, err
	}

	items := make([]models.BatchItem, 0, len(input.Prompts))
	for index, prompt := range input.Prompts {
		var item models.BatchItem
		err = tx.QueryRow(ctx, `
			INSERT INTO batch_items (batch_job_id, prompt_index, prompt_text, status)
			VALUES ($1, $2, $3, 'queued')
			RETURNING id::text, prompt_index, prompt_text, status, created_at, updated_at
		`, batch.ID, index+1, prompt).Scan(
			&item.ID,
			&item.PromptIndex,
			&item.PromptText,
			&item.Status,
			&item.CreatedAt,
			&item.UpdatedAt,
		)
		if err != nil {
			return models.BatchJob{}, err
		}
		items = append(items, item)
	}

	batch.Items = items
	if err := tx.Commit(ctx); err != nil {
		return models.BatchJob{}, err
	}

	return r.GetBatch(ctx, batch.ID)
}

func (r *Repository) GetBatch(ctx context.Context, id string) (models.BatchJob, error) {
	var batch models.BatchJob
	err := r.db.QueryRow(ctx, `
		SELECT id::text, character_set_id::text, title, global_style, status, width, height, total_count, completed_count, failed_count, created_at, updated_at
		FROM batch_jobs
		WHERE id = $1
	`, id).Scan(
		&batch.ID,
		&batch.CharacterSetID,
		&batch.Title,
		&batch.GlobalStyle,
		&batch.Status,
		&batch.Width,
		&batch.Height,
		&batch.TotalCount,
		&batch.CompletedCount,
		&batch.FailedCount,
		&batch.CreatedAt,
		&batch.UpdatedAt,
	)
	if err != nil {
		return batch, err
	}

	set, err := r.GetCharacterSet(ctx, batch.CharacterSetID)
	if err != nil {
		return batch, err
	}
	batch.CharacterSet = set

	rows, err := r.db.Query(ctx, `
		SELECT id::text, prompt_index, prompt_text, status, COALESCE(image_key, ''), COALESCE(error, ''), created_at, updated_at
		FROM batch_items
		WHERE batch_job_id = $1
		ORDER BY prompt_index ASC
	`, id)
	if err != nil {
		return batch, err
	}
	defer rows.Close()

	batch.Items = []models.BatchItem{}
	for rows.Next() {
		var item models.BatchItem
		var imageKey string
		if err := rows.Scan(&item.ID, &item.PromptIndex, &item.PromptText, &item.Status, &imageKey, &item.Error, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return batch, err
		}
		if imageKey != "" {
			item.ImageURL = "/outputs/" + imageKey
		}
		batch.Items = append(batch.Items, item)
	}
	return batch, rows.Err()
}

func (r *Repository) MarkBatchRunning(ctx context.Context, batchID string) error {
	_, err := r.db.Exec(ctx, `UPDATE batch_jobs SET status = 'running', updated_at = NOW() WHERE id = $1`, batchID)
	return err
}

func (r *Repository) MarkBatchItemRunning(ctx context.Context, itemID string) error {
	_, err := r.db.Exec(ctx, `UPDATE batch_items SET status = 'running', updated_at = NOW() WHERE id = $1`, itemID)
	return err
}

func (r *Repository) MarkBatchItemSucceeded(ctx context.Context, batchID, itemID, imageKey string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `UPDATE batch_items SET status = 'succeeded', image_key = $2, error = '', updated_at = NOW() WHERE id = $1`, itemID, imageKey); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE batch_jobs SET completed_count = completed_count + 1, updated_at = NOW() WHERE id = $1`, batchID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE batch_jobs
		SET status = CASE WHEN completed_count + failed_count + 1 >= total_count THEN 'completed' ELSE status END,
		    updated_at = NOW()
		WHERE id = $1
	`, batchID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *Repository) MarkBatchItemFailed(ctx context.Context, batchID, itemID, errorMessage string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `UPDATE batch_items SET status = 'failed', error = $2, updated_at = NOW() WHERE id = $1`, itemID, errorMessage); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE batch_jobs SET failed_count = failed_count + 1, updated_at = NOW() WHERE id = $1`, batchID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE batch_jobs
		SET status = CASE WHEN completed_count + failed_count + 1 >= total_count THEN 'completed' ELSE status END,
		    updated_at = NOW()
		WHERE id = $1
	`, batchID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *Repository) FailBatch(ctx context.Context, batchID, errorMessage string) error {
	_, err := r.db.Exec(ctx, `UPDATE batch_jobs SET status = 'failed', updated_at = NOW() WHERE id = $1`, batchID)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(ctx, `
		UPDATE batch_items
		SET status = CASE WHEN status IN ('queued', 'running') THEN 'failed' ELSE status END,
		    error = CASE WHEN status IN ('queued', 'running') THEN $2 ELSE error END,
		    updated_at = NOW()
		WHERE batch_job_id = $1
	`, batchID, errorMessage)
	return err
}
