package services

import (
	"strings"
	"whisk-clone/adapters"
)

var presetSuffixes = map[string]string{
	"photorealistic": "ultra detailed, 8k, photorealistic, DSLR",
	"cinematic":      "cinematic lighting, movie still, anamorphic lens",
	"anime":          "anime style, studio ghibli, detailed illustration",
	"oil_painting":   "oil on canvas, impressionist brushwork",
	"watercolor":     "watercolor painting, soft edges, translucent",
	"pixel_art":      "pixel art, 16-bit, retro game style",
	"sketched":       "pencil sketch, hand drawn, monochrome",
}

type GeneratorService struct {
	adapter *adapters.TogetherAI
	storage *Storage
}

func NewGeneratorService(adapter *adapters.TogetherAI, storage *Storage) *GeneratorService {
	return &GeneratorService{adapter: adapter, storage: storage}
}

func (g *GeneratorService) BuildPrompt(subject, scene, style, preset string) string {
	parts := []string{}
	if subject != "" {
		parts = append(parts, subject)
	}
	if scene != "" {
		parts = append(parts, scene)
	}
	if style != "" {
		parts = append(parts, style)
	}
	master := strings.Join(parts, ", ")
	if suffix, ok := presetSuffixes[preset]; ok {
		master += ", " + suffix
	}
	return master
}

func (g *GeneratorService) Generate(subject, scene, style, preset string) (string, error) {
	prompt := g.BuildPrompt(subject, scene, style, preset)
	imgBytes, err := g.adapter.GenerateImage(prompt)
	if err != nil {
		return "", err
	}
	filename, err := g.storage.SaveImage(imgBytes, "gen")
	if err != nil {
		return "", err
	}
	return filename, nil
}
