package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"whisk-clone/adapters"
	"whisk-clone/models"
	"whisk-clone/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func RefineHandler(adapter *adapters.TogetherAI, generator *services.GeneratorService, storage *adapters.StorageAdapter) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req models.RefineRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// 1. Refine prompts via Cloudflare Workers AI (Llama 3.1 8B)
		newSubject, newScene, newStyle, err := adapter.RefinePrompts(
			req.SubjectPrompt, req.ScenePrompt, req.StylePrompt, req.Feedback,
		)
		if err != nil {
			log.Printf("RefinePrompts error: %v — using original prompts", err)
			newSubject, newScene, newStyle = req.SubjectPrompt, req.ScenePrompt, req.StylePrompt
		}

		// For FLUX.2 reference-guided editing: explicitly tell the model to preserve the person from image 0
		refinedPrompt := "The exact same person from image 0, " + generator.BuildPrompt(newSubject, newScene, newStyle, req.StylePreset)

		var imgBytes []byte

		// 2. If original image URL provided, do img2img (preserves structure)
		if req.OriginalURL != "" && storage != nil {
			// Extract storage key from /outputs/gen_xxx.png
			key := strings.TrimPrefix(req.OriginalURL, "/outputs/")
			origBytes, err := storage.Download(c.Request.Context(), key)
			if err == nil {
				imgBytes, err = adapter.Img2Img(origBytes, refinedPrompt, 0.65)
				if err != nil {
					log.Printf("Img2Img error: %v — falling back to text2img", err)
					imgBytes = nil
				}
			} else {
				log.Printf("Download original image error: %v — falling back to text2img", err)
			}
		}

		// 3. Fallback to text2img if img2img not available or failed
		if imgBytes == nil {
			width, height := req.Width, req.Height
			if width <= 0 {
				width = 1024
			}
			if height <= 0 {
				height = 1024
			}
			imgBytes, err = adapter.GenerateImage(refinedPrompt, width, height)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		}

		// 4. Save to storage and DB
		key := fmt.Sprintf("gen_%s.png", uuid.New().String()[:8])
		userID := c.GetString("user_id")

		if storage != nil {
			if err := storage.Upload(c.Request.Context(), key, imgBytes, "image/png"); err != nil {
				log.Printf("Storage upload error: %v", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "storage upload failed"})
				return
			}
			if userID != "" {
				if err := generator.SaveImageRecord(c.Request.Context(), userID, key, newSubject, newScene, newStyle, req.StylePreset, req.Width, req.Height); err != nil {
					log.Printf("SaveImageRecord error: %v", err)
				}
			}
		}

		c.JSON(http.StatusOK, models.RefineResponse{
			ImageURL:      "/outputs/" + key,
			SubjectPrompt: newSubject,
			ScenePrompt:   newScene,
			StylePrompt:   newStyle,
		})
	}
}
