package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

const defaultPageSize = 20

func UserImagesHandler(dbPool *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		if dbPool == nil {
			c.JSON(http.StatusOK, gin.H{"images": []interface{}{}, "total": 0, "page": 1, "page_size": defaultPageSize, "has_next": false})
			return
		}

		userID := c.GetString("user_id")

		page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
		pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", strconv.Itoa(defaultPageSize)))
		if page < 1 {
			page = 1
		}
		if pageSize < 1 || pageSize > 100 {
			pageSize = defaultPageSize
		}
		offset := (page - 1) * pageSize

		var total int
		if err := dbPool.QueryRow(c.Request.Context(),
			`SELECT COUNT(*) FROM generated_images WHERE user_id = $1`, userID,
		).Scan(&total); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to count images"})
			return
		}

		rows, err := dbPool.Query(c.Request.Context(),
			`SELECT id, storage_key, subject_prompt, scene_prompt, style_prompt, style_preset, width, height, created_at
			 FROM generated_images
			 WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
			userID, pageSize, offset,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch images"})
			return
		}
		defer rows.Close()

		type ImageItem struct {
			ID            string    `json:"id"`
			URL           string    `json:"url"`
			SubjectPrompt string    `json:"subject_prompt"`
			ScenePrompt   string    `json:"scene_prompt"`
			StylePrompt   string    `json:"style_prompt"`
			StylePreset   string    `json:"style_preset"`
			Width         int       `json:"width"`
			Height        int       `json:"height"`
			CreatedAt     time.Time `json:"created_at"`
		}

		var images []ImageItem
		for rows.Next() {
			var item ImageItem
			var key string
			if err := rows.Scan(&item.ID, &key, &item.SubjectPrompt, &item.ScenePrompt, &item.StylePrompt, &item.StylePreset, &item.Width, &item.Height, &item.CreatedAt); err != nil {
				continue
			}
			item.URL = "/outputs/" + key
			images = append(images, item)
		}
		if images == nil {
			images = []ImageItem{}
		}

		c.JSON(http.StatusOK, gin.H{
			"images":    images,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
			"has_next":  offset+pageSize < total,
		})
	}
}
