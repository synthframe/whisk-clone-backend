package adapters

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type WorkerClient struct {
	BaseURL    string
	HTTPClient *http.Client
}

func NewWorkerClient(baseURL string) *WorkerClient {
	return &WorkerClient{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

func (c *WorkerClient) GenerateImage(prompt string, width, height int, referenceImages [][]byte) ([]byte, string, error) {
	encoded := make([]string, 0, len(referenceImages))
	for _, image := range referenceImages {
		if len(image) == 0 {
			continue
		}
		encoded = append(encoded, base64.StdEncoding.EncodeToString(image))
	}

	payload, err := json.Marshal(map[string]any{
		"prompt":           prompt,
		"width":            width,
		"height":           height,
		"reference_images": encoded,
	})
	if err != nil {
		return nil, "", err
	}

	resp, err := c.HTTPClient.Post(c.BaseURL, "application/json", bytes.NewReader(payload))
	if err != nil {
		return nil, "", fmt.Errorf("worker request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read worker response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("worker error %d: %s", resp.StatusCode, string(body))
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = http.DetectContentType(body)
	}

	return body, contentType, nil
}
