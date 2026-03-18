package main

import (
	"log"
	"whisk-clone/adapters"
	"whisk-clone/config"
	"whisk-clone/handlers"
	"whisk-clone/services"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()
	if cfg.TogetherAPIKey == "" {
		log.Println("WARNING: TOGETHER_API_KEY is not set")
	}

	storage, err := services.NewStorage(cfg.OutputDir)
	if err != nil {
		log.Fatalf("failed to init storage: %v", err)
	}

	togetherAdapter := adapters.NewTogetherAI(cfg.TogetherAPIKey)
	visionSvc := services.NewVisionService(togetherAdapter)
	generatorSvc := services.NewGeneratorService(togetherAdapter, storage)
	batchSvc := services.NewBatchService(generatorSvc)

	r := gin.Default()
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: false,
	}))

	// Static file serving for generated images (no-store to prevent stale cache)
	r.GET("/outputs/:filename", func(c *gin.Context) {
		c.Header("Cache-Control", "no-store")
		c.File(cfg.OutputDir + "/" + c.Param("filename"))
	})

	api := r.Group("/api")
	{
		api.GET("/health", handlers.HealthHandler(cfg))
		api.POST("/analyze", handlers.AnalyzeHandler(visionSvc))
		api.POST("/generate", handlers.GenerateHandler(generatorSvc))
		api.GET("/generate/:id", handlers.GenerateStatusHandler())
		api.POST("/batch", handlers.BatchCreateHandler(batchSvc))
		api.GET("/batch/:id", handlers.BatchStatusHandler())
		api.GET("/batch/:id/stream", handlers.BatchStreamHandler())
	}

	log.Printf("Server starting on %s", cfg.ServerPort)
	if err := r.Run(cfg.ServerPort); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
