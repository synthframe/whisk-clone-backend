package services

import (
	"encoding/base64"
	"fmt"
	"io"
	"whisk-clone/adapters"
)

type VisionService struct {
	adapter *adapters.TogetherAI
}

func NewVisionService(adapter *adapters.TogetherAI) *VisionService {
	return &VisionService{adapter: adapter}
}

func (v *VisionService) AnalyzeImage(imageReader io.Reader, slotType string) (string, error) {
	imgBytes, err := io.ReadAll(imageReader)
	if err != nil {
		return "", fmt.Errorf("failed to read image: %w", err)
	}
	b64 := base64.StdEncoding.EncodeToString(imgBytes)
	return v.adapter.AnalyzeImage(b64, slotType)
}
