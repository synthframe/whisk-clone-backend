package models

import "time"

// Analyze
type AnalyzeRequest struct {
	SlotType string `form:"slot_type" binding:"required"`
}

type AnalyzeResponse struct {
	Prompt string `json:"prompt"`
}

// Generate
type GenerateRequest struct {
	SubjectPrompt string `json:"subject_prompt"`
	ScenePrompt   string `json:"scene_prompt"`
	StylePrompt   string `json:"style_prompt"`
	StylePreset   string `json:"style_preset"`
}

type GenerateResponse struct {
	ID       string `json:"id"`
	Status   string `json:"status"`
	ImageURL string `json:"image_url,omitempty"`
	Error    string `json:"error,omitempty"`
}

// Batch
type BatchJobInput struct {
	SubjectPrompt string `json:"subject_prompt"`
	ScenePrompt   string `json:"scene_prompt"`
	StylePrompt   string `json:"style_prompt"`
	StylePreset   string `json:"style_preset"`
}

type BatchRequest struct {
	Jobs        []BatchJobInput `json:"jobs" binding:"required"`
	Concurrency int             `json:"concurrency"`
}

type BatchResponse struct {
	BatchID string `json:"batch_id"`
	Total   int    `json:"total"`
}

type BatchJobResult struct {
	Index    int    `json:"index"`
	Status   string `json:"status"`
	ImageURL string `json:"image_url,omitempty"`
	Error    string `json:"error,omitempty"`
}

type BatchStatusResponse struct {
	BatchID   string           `json:"batch_id"`
	Status    string           `json:"status"`
	Total     int              `json:"total"`
	Completed int              `json:"completed"`
	Failed    int              `json:"failed"`
	Results   []BatchJobResult `json:"results"`
}

// SSE
type SSEEvent struct {
	Type      string      `json:"type"`
	Timestamp time.Time   `json:"timestamp"`
	Payload   interface{} `json:"payload"`
}

// Health
type HealthResponse struct {
	Status  string `json:"status"`
	Model   string `json:"model"`
	APIKey  bool   `json:"api_key_set"`
}
