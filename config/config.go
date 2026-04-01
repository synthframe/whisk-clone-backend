package config

import "os"

type Config struct {
	ServerPort    string
	DatabaseURL   string
	S3Endpoint    string
	S3AccessKey   string
	S3SecretKey   string
	S3Bucket      string
	WorkerBaseURL string
}

func Load() *Config {
	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = ":8080"
	}
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgresql://postgres@whisk-db:5432/whisk_db?sslmode=disable"
	}
	s3Endpoint := os.Getenv("S3_ENDPOINT")
	if s3Endpoint == "" {
		s3Endpoint = "http://whisk-storage:8333"
	}
	s3Bucket := os.Getenv("S3_BUCKET")
	if s3Bucket == "" {
		s3Bucket = "uploads"
	}
	workerBaseURL := os.Getenv("WORKER_BASE_URL")
	if workerBaseURL == "" {
		workerBaseURL = "https://whisk-image-gen.gimchan29.workers.dev"
	}

	return &Config{
		ServerPort:    port,
		DatabaseURL:   dbURL,
		S3Endpoint:    s3Endpoint,
		S3AccessKey:   os.Getenv("S3_ACCESS_KEY"),
		S3SecretKey:   os.Getenv("S3_SECRET_KEY"),
		S3Bucket:      s3Bucket,
		WorkerBaseURL: workerBaseURL,
	}
}
