package handlers

import (
	"net/http"
	"whisk-clone/models"
	"whisk-clone/services"

	"github.com/gin-gonic/gin"
)

func AnalyzeHandler(vision *services.VisionService) gin.HandlerFunc {
	return func(c *gin.Context) {
		slotType := c.PostForm("slot_type")
		if slotType == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "slot_type is required"})
			return
		}

		file, err := c.FormFile("image")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "image file is required"})
			return
		}

		f, err := file.Open()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to open image"})
			return
		}
		defer f.Close()

		prompt, err := vision.AnalyzeImage(f, slotType)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, models.AnalyzeResponse{Prompt: prompt})
	}
}
