package models

import "time"

type CharacterReference struct {
	ID         string    `json:"id"`
	StorageKey string    `json:"storage_key"`
	ImageURL   string    `json:"image_url"`
	CreatedAt  time.Time `json:"created_at"`
}

type CharacterSet struct {
	ID          string               `json:"id"`
	Name        string               `json:"name"`
	Description string               `json:"description"`
	GlobalStyle string               `json:"global_style"`
	CreatedAt   time.Time            `json:"created_at"`
	References  []CharacterReference `json:"references"`
}

type BatchItem struct {
	ID          string    `json:"id"`
	PromptIndex int       `json:"prompt_index"`
	PromptText  string    `json:"prompt_text"`
	Status      string    `json:"status"`
	ImageURL    string    `json:"image_url,omitempty"`
	Error       string    `json:"error,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type BatchJob struct {
	ID             string       `json:"id"`
	CharacterSetID string       `json:"character_set_id"`
	CharacterSet   CharacterSet `json:"character_set"`
	Title          string       `json:"title"`
	GlobalStyle    string       `json:"global_style"`
	Status         string       `json:"status"`
	Width          int          `json:"width"`
	Height         int          `json:"height"`
	TotalCount     int          `json:"total_count"`
	CompletedCount int          `json:"completed_count"`
	FailedCount    int          `json:"failed_count"`
	CreatedAt      time.Time    `json:"created_at"`
	UpdatedAt      time.Time    `json:"updated_at"`
	Items          []BatchItem  `json:"items"`
}

type CreateCharacterSetInput struct {
	Name        string
	Description string
	GlobalStyle string
}

type CreateBatchInput struct {
	CharacterSetID string
	Title          string
	GlobalStyle    string
	Prompts        []string
	Width          int
	Height         int
}
