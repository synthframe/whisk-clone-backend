package services

import (
	"sync"
	"time"
	"whisk-clone/models"
)

type BatchSession struct {
	ID          string
	Jobs        []models.BatchJobInput
	Results     []models.BatchJobResult
	Events      chan models.SSEEvent
	Status      string
	mu          sync.Mutex
	completedCh chan struct{}
}

var batchSessions sync.Map

type BatchService struct {
	generator *GeneratorService
}

func NewBatchService(generator *GeneratorService) *BatchService {
	return &BatchService{generator: generator}
}

func (b *BatchService) CreateSession(id string, jobs []models.BatchJobInput, concurrency int) *BatchSession {
	bufSize := len(jobs) * 3
	if bufSize < 10 {
		bufSize = 10
	}
	session := &BatchSession{
		ID:          id,
		Jobs:        jobs,
		Results:     make([]models.BatchJobResult, len(jobs)),
		Events:      make(chan models.SSEEvent, bufSize),
		Status:      "running",
		completedCh: make(chan struct{}),
	}
	batchSessions.Store(id, session)
	go b.run(session, concurrency)
	return session
}

func GetSession(id string) (*BatchSession, bool) {
	v, ok := batchSessions.Load(id)
	if !ok {
		return nil, false
	}
	return v.(*BatchSession), true
}

func (b *BatchService) run(session *BatchSession, concurrency int) {
	if concurrency <= 0 {
		concurrency = 2
	}
	sem := make(chan struct{}, concurrency)

	// heartbeat goroutine
	stopHB := make(chan struct{})
	go func() {
		t := time.NewTicker(15 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-t.C:
				session.Events <- models.SSEEvent{
					Type:      "heartbeat",
					Timestamp: time.Now(),
					Payload:   map[string]string{"status": "alive"},
				}
			case <-stopHB:
				return
			}
		}
	}()

	var wg sync.WaitGroup
	for i, job := range session.Jobs {
		wg.Add(1)
		i, job := i, job
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()

			session.Events <- models.SSEEvent{
				Type:      "job_started",
				Timestamp: time.Now(),
				Payload:   map[string]interface{}{"index": i},
			}

			filename, err := b.generator.Generate(job.SubjectPrompt, job.ScenePrompt, job.StylePrompt, job.StylePreset)

			session.mu.Lock()
			if err != nil {
				session.Results[i] = models.BatchJobResult{
					Index:  i,
					Status: "failed",
					Error:  err.Error(),
				}
				session.Events <- models.SSEEvent{
					Type:      "job_failed",
					Timestamp: time.Now(),
					Payload:   map[string]interface{}{"index": i, "error": err.Error()},
				}
			} else {
				session.Results[i] = models.BatchJobResult{
					Index:    i,
					Status:   "completed",
					ImageURL: "/outputs/" + filename,
				}
				session.Events <- models.SSEEvent{
					Type:      "job_completed",
					Timestamp: time.Now(),
					Payload:   map[string]interface{}{"index": i, "image_url": "/outputs/" + filename},
				}
			}
			session.mu.Unlock()
		}()
	}

	wg.Wait()
	close(stopHB)

	session.mu.Lock()
	session.Status = "completed"
	session.mu.Unlock()

	session.Events <- models.SSEEvent{
		Type:      "batch_completed",
		Timestamp: time.Now(),
		Payload:   map[string]interface{}{"batch_id": session.ID, "total": len(session.Jobs)},
	}
	close(session.Events)
}

func (s *BatchSession) GetStatus() models.BatchStatusResponse {
	s.mu.Lock()
	defer s.mu.Unlock()

	completed, failed := 0, 0
	for _, r := range s.Results {
		switch r.Status {
		case "completed":
			completed++
		case "failed":
			failed++
		}
	}
	results := make([]models.BatchJobResult, len(s.Results))
	copy(results, s.Results)
	return models.BatchStatusResponse{
		BatchID:   s.ID,
		Status:    s.Status,
		Total:     len(s.Jobs),
		Completed: completed,
		Failed:    failed,
		Results:   results,
	}
}
