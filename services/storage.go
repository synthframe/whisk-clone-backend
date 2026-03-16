package services

import (
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
)

type Storage struct {
	OutputDir string
	counter   atomic.Int64
}

func NewStorage(outputDir string) (*Storage, error) {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output dir: %w", err)
	}
	s := &Storage{OutputDir: outputDir}
	// Initialize counter based on existing files
	entries, _ := os.ReadDir(outputDir)
	s.counter.Store(int64(len(entries)))
	return s, nil
}

func (s *Storage) SaveImage(data []byte, prefix string) (string, error) {
	idx := s.counter.Add(1)
	filename := fmt.Sprintf("%s_%03d.png", prefix, idx)
	fullPath := filepath.Join(s.OutputDir, filename)
	if err := os.WriteFile(fullPath, data, 0644); err != nil {
		return "", fmt.Errorf("failed to save image: %w", err)
	}
	return filename, nil
}
