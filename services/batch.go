package services

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"synthframe-api/adapters"
)

type BatchProcessor struct {
	repo    *Repository
	storage *adapters.StorageAdapter
	worker  *adapters.WorkerClient

	mu      sync.Mutex
	running map[string]bool
}

func NewBatchProcessor(repo *Repository, storage *adapters.StorageAdapter, worker *adapters.WorkerClient) *BatchProcessor {
	return &BatchProcessor{
		repo:    repo,
		storage: storage,
		worker:  worker,
		running: map[string]bool{},
	}
}

func (p *BatchProcessor) Start(batchID string) {
	p.mu.Lock()
	if p.running[batchID] {
		p.mu.Unlock()
		return
	}
	p.running[batchID] = true
	p.mu.Unlock()

	go func() {
		defer func() {
			p.mu.Lock()
			delete(p.running, batchID)
			p.mu.Unlock()
		}()

		ctx := context.Background()
		if err := p.process(ctx, batchID); err != nil {
			_ = p.repo.FailBatch(ctx, batchID, err.Error())
		}
	}()
}

func (p *BatchProcessor) process(ctx context.Context, batchID string) error {
	batch, err := p.repo.GetBatch(ctx, batchID)
	if err != nil {
		return err
	}
	if err := p.repo.MarkBatchRunning(ctx, batchID); err != nil {
		return err
	}

	referenceImages := make([][]byte, 0, len(batch.CharacterSet.References))
	for _, reference := range batch.CharacterSet.References {
		image, err := p.storage.Download(ctx, reference.StorageKey)
		if err != nil {
			return fmt.Errorf("failed to load reference image %s: %w", reference.StorageKey, err)
		}
		referenceImages = append(referenceImages, image)
	}

	for _, item := range batch.Items {
		if err := p.repo.MarkBatchItemRunning(ctx, item.ID); err != nil {
			return err
		}

		prompt := buildPrompt(batch.CharacterSet.Name, batch.CharacterSet.Description, batch.CharacterSet.GlobalStyle, batch.GlobalStyle, item.PromptText)
		imageBytes, contentType, err := p.worker.GenerateImage(prompt, batch.Width, batch.Height, referenceImages)
		if err != nil {
			if markErr := p.repo.MarkBatchItemFailed(ctx, batchID, item.ID, err.Error()); markErr != nil {
				return markErr
			}
			continue
		}

		ext := ".jpg"
		if strings.Contains(contentType, "png") {
			ext = ".png"
		}
		key := fmt.Sprintf("batch_%s_%02d%s", batchID[:8], item.PromptIndex, ext)
		if err := p.storage.Upload(ctx, key, imageBytes, contentType); err != nil {
			if markErr := p.repo.MarkBatchItemFailed(ctx, batchID, item.ID, err.Error()); markErr != nil {
				return markErr
			}
			continue
		}

		if err := p.repo.MarkBatchItemSucceeded(ctx, batchID, item.ID, key); err != nil {
			return err
		}
	}

	return nil
}

func buildPrompt(name, description, setStyle, batchStyle, shotPrompt string) string {
	parts := []string{
		"Create a production-ready image for a persistent character set.",
		"Keep the same identity, face, proportions, and overall vibe from the reference images.",
	}
	if name != "" {
		parts = append(parts, "Character name: "+name+".")
	}
	if description != "" {
		parts = append(parts, "Character profile: "+description+".")
	}
	if setStyle != "" {
		parts = append(parts, "Character set style: "+setStyle+".")
	}
	if batchStyle != "" {
		parts = append(parts, "Global batch style: "+batchStyle+".")
	}
	parts = append(parts, "Shot requirement: "+shotPrompt+".")
	parts = append(parts, "High quality image, detailed face, coherent lighting, strong commercial finish.")
	return strings.Join(parts, " ")
}
