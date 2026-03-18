package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
	"whisk-clone/models"
	"whisk-clone/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func BatchCreateHandler(batchSvc *services.BatchService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req models.BatchRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if len(req.Jobs) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "jobs list is empty"})
			return
		}

		batchID := uuid.New().String()
		userID := c.GetString("user_id")
		batchSvc.CreateSession(batchID, userID, req.Jobs, req.Concurrency)

		c.JSON(http.StatusAccepted, models.BatchResponse{
			BatchID: batchID,
			Total:   len(req.Jobs),
		})
	}
}

func BatchStatusHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		session, ok := services.GetSession(id)
		if !ok {
			c.JSON(http.StatusNotFound, gin.H{"error": "batch not found"})
			return
		}
		c.JSON(http.StatusOK, session.GetStatus())
	}
}

func BatchStreamHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		session, ok := services.GetSession(id)
		if !ok {
			c.JSON(http.StatusNotFound, gin.H{"error": "batch not found"})
			return
		}

		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")
		c.Header("X-Accel-Buffering", "no")

		flusher, ok := c.Writer.(http.Flusher)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "streaming not supported"})
			return
		}

		clientGone := c.Request.Context().Done()

		for {
			select {
			case <-clientGone:
				return
			case event, open := <-session.Events:
				if !open {
					fmt.Fprintf(c.Writer, "event: done\ndata: {}\n\n")
					flusher.Flush()
					return
				}
				data, _ := json.Marshal(event)
				fmt.Fprintf(c.Writer, "data: %s\n\n", string(data))
				flusher.Flush()
			case <-time.After(30 * time.Second):
				// timeout waiting for events - check if batch is done
				session2, exists := services.GetSession(id)
				if !exists || session2.GetStatus().Status == "completed" {
					fmt.Fprintf(c.Writer, "event: done\ndata: {}\n\n")
					flusher.Flush()
					return
				}
			}
		}
	}
}
