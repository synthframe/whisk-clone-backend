package main

import (
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"

	"synthframe-api/adapters"
	"synthframe-api/config"
	"synthframe-api/db"
	"synthframe-api/models"
	"synthframe-api/services"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func main() {
	cfg := config.Load()

	dbPool, err := db.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	if err := db.Migrate(dbPool); err != nil {
		log.Fatalf("failed to migrate database: %v", err)
	}

	storage := adapters.NewStorageAdapter(cfg.S3Endpoint, cfg.S3AccessKey, cfg.S3SecretKey, cfg.S3Bucket)
	worker := adapters.NewWorkerClient(cfg.WorkerBaseURL)
	repo := services.NewRepository(dbPool)
	batchProcessor := services.NewBatchProcessor(repo, storage, worker)

	r := gin.Default()
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: false,
	}))

	r.GET("/outputs/:filename", func(c *gin.Context) {
		filename := c.Param("filename")
		data, err := storage.Download(c.Request.Context(), filename)
		if err != nil {
			c.Status(http.StatusNotFound)
			return
		}
		c.Header("Cache-Control", "public, max-age=86400")
		contentType := "image/jpeg"
		if strings.HasSuffix(filename, ".png") {
			contentType = "image/png"
		} else if strings.HasSuffix(filename, ".webp") {
			contentType = "image/webp"
		}
		c.Data(http.StatusOK, contentType, data)
	})

	api := r.Group("/api")
	api.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "synthframe-api"})
	})

	api.GET("/character-sets", func(c *gin.Context) {
		sets, err := repo.ListCharacterSets(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"character_sets": sets})
	})

	api.GET("/character-sets/:id", func(c *gin.Context) {
		set, err := repo.GetCharacterSet(c.Request.Context(), c.Param("id"))
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "character set not found"})
			return
		}
		c.JSON(http.StatusOK, set)
	})

	api.DELETE("/character-sets/:id", func(c *gin.Context) {
		set, err := repo.GetCharacterSet(c.Request.Context(), c.Param("id"))
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "character set not found"})
			return
		}

		for _, reference := range set.References {
			if deleteErr := storage.Delete(c.Request.Context(), reference.StorageKey); deleteErr != nil {
				log.Printf("warning: failed to delete reference image %s: %v", reference.StorageKey, deleteErr)
			}
		}

		if err := repo.DeleteCharacterSet(c.Request.Context(), set.ID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.Status(http.StatusNoContent)
	})

	api.POST("/character-sets", func(c *gin.Context) {
		name := strings.TrimSpace(c.PostForm("name"))
		if name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
			return
		}

		set, err := repo.CreateCharacterSet(c.Request.Context(), models.CreateCharacterSetInput{
			Name:        name,
			Description: strings.TrimSpace(c.PostForm("description")),
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		form, err := c.MultipartForm()
		if err == nil && form != nil {
			for index, fileHeader := range form.File["references"] {
				key, uploadErr := saveReferenceFile(c, storage, set.ID, index, fileHeader)
				if uploadErr != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": uploadErr.Error()})
					return
				}
				if err := repo.AddCharacterReference(c.Request.Context(), set.ID, key); err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}
			}
		}

		created, err := repo.GetCharacterSet(c.Request.Context(), set.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, created)
	})

	api.POST("/batches", func(c *gin.Context) {
		var payload struct {
			CharacterSetID string   `json:"character_set_id"`
			Title          string   `json:"title"`
			GlobalStyle    string   `json:"global_style"`
			Prompts        []string `json:"prompts"`
			Width          int      `json:"width"`
			Height         int      `json:"height"`
		}
		if err := c.ShouldBindJSON(&payload); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if payload.CharacterSetID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "character_set_id is required"})
			return
		}

		prompts := make([]string, 0, len(payload.Prompts))
		for _, prompt := range payload.Prompts {
			if trimmed := strings.TrimSpace(prompt); trimmed != "" {
				prompts = append(prompts, trimmed)
			}
		}
		if len(prompts) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "at least one prompt is required"})
			return
		}
		if payload.Width == 0 {
			payload.Width = 1024
		}
		if payload.Height == 0 {
			payload.Height = 1024
		}

		batch, err := repo.CreateBatch(c.Request.Context(), models.CreateBatchInput{
			CharacterSetID: payload.CharacterSetID,
			Title:          strings.TrimSpace(payload.Title),
			GlobalStyle:    strings.TrimSpace(payload.GlobalStyle),
			Prompts:        prompts,
			Width:          payload.Width,
			Height:         payload.Height,
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		batchProcessor.Start(batch.ID)
		c.JSON(http.StatusAccepted, batch)
	})

	api.GET("/batches/:id", func(c *gin.Context) {
		batch, err := repo.GetBatch(c.Request.Context(), c.Param("id"))
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "batch not found"})
			return
		}
		c.JSON(http.StatusOK, batch)
	})

	log.Printf("server starting on %s", cfg.ServerPort)
	if err := r.Run(cfg.ServerPort); err != nil {
		log.Fatal(err)
	}
}

func saveReferenceFile(c *gin.Context, storage *adapters.StorageAdapter, characterSetID string, index int, fileHeader *multipart.FileHeader) (string, error) {
	file, err := fileHeader.Open()
	if err != nil {
		return "", err
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return "", err
	}

	contentType := fileHeader.Header.Get("Content-Type")
	if contentType == "" {
		contentType = http.DetectContentType(data)
	}

	ext := ".jpg"
	if strings.Contains(contentType, "png") {
		ext = ".png"
	} else if strings.Contains(contentType, "webp") {
		ext = ".webp"
	}

	key := fmt.Sprintf("character_%s_%02d_%s%s", characterSetID[:8], index+1, uuid.NewString()[:8], ext)
	if err := storage.Upload(c.Request.Context(), key, data, contentType); err != nil {
		return "", err
	}
	return filepath.Base(key), nil
}
