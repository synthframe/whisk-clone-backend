package config

import (
	"os"
)

type Config struct {
	TogetherAPIKey string
	OutputDir      string
	ServerPort     string
}

func Load() *Config {
	outputDir := os.Getenv("OUTPUT_DIR")
	if outputDir == "" {
		outputDir = "./outputs"
	}
	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = ":8080"
	}
	return &Config{
		TogetherAPIKey: os.Getenv("TOGETHER_API_KEY"),
		OutputDir:      outputDir,
		ServerPort:     port,
	}
}
