package middleware

import (
	"net/http"
	"strings"
	"whisk-clone/services"

	"github.com/gin-gonic/gin"
)

func Auth(authSvc *services.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// SSE endpoints can't send headers — also accept ?token= query param
		tokenStr := c.Query("token")
		if tokenStr == "" {
			header := c.GetHeader("Authorization")
			if header == "" || !strings.HasPrefix(header, "Bearer ") {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "인증이 필요합니다"})
				c.Abort()
				return
			}
			tokenStr = strings.TrimPrefix(header, "Bearer ")
		}
		userID, err := authSvc.ValidateToken(tokenStr)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "유효하지 않은 토큰입니다"})
			c.Abort()
			return
		}

		c.Set("user_id", userID)
		c.Next()
	}
}
