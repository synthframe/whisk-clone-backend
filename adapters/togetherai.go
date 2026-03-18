package adapters

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type TogetherAI struct {
	APIKey     string
	HTTPClient *http.Client
}

func NewTogetherAI(apiKey string) *TogetherAI {
	return &TogetherAI{
		APIKey:     apiKey,
		HTTPClient: &http.Client{},
	}
}

// Vision: analyze image via Cloudflare Workers AI (LLaVA 1.5)
func (t *TogetherAI) AnalyzeImage(imageBase64 string, slotType string) (string, error) {
	payload, err := json.Marshal(map[string]interface{}{
		"image_base64": imageBase64,
		"slot_type":    slotType,
	})
	if err != nil {
		return "", err
	}

	resp, err := t.HTTPClient.Post(
		"https://whisk-image-gen.gimchan29.workers.dev/analyze",
		"application/json",
		bytes.NewReader(payload),
	)
	if err != nil {
		return "", fmt.Errorf("cloudflare worker vision request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("cloudflare worker vision error %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Prompt string `json:"prompt"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if result.Prompt == "" {
		return "", fmt.Errorf("empty prompt from vision API")
	}
	return result.Prompt, nil
}

// GenerateImage: generate image via Cloudflare Workers AI (free, no credits required)
func (t *TogetherAI) GenerateImage(prompt string) ([]byte, error) {
	payload, err := json.Marshal(map[string]interface{}{
		"prompt": prompt,
		"width":  1024,
		"height": 1024,
	})
	if err != nil {
		return nil, err
	}

	resp, err := t.HTTPClient.Post(
		"https://whisk-image-gen.gimchan29.workers.dev",
		"application/json",
		bytes.NewReader(payload),
	)
	if err != nil {
		return nil, fmt.Errorf("cloudflare worker request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("cloudflare worker error %d: %s", resp.StatusCode, string(respBody))
	}

	imgBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read image bytes: %w", err)
	}
	return imgBytes, nil
}

func slotPrompt(slotType string) string {
	switch slotType {
	case "subject":
		return "Describe only the main subject/character. Be concise, comma-separated."
	case "scene":
		return "Describe only the setting/background environment. Be concise."
	case "style":
		return "Describe only the artistic style, color palette, and lighting. Be concise."
	default:
		return "Describe the image concisely."
	}
}
