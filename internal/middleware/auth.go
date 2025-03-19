package middleware

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
)

func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString, err := c.Cookie("token")
		if err != nil {
			handleUnauthorized(c)
			return
		}

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return []byte(os.Getenv("JWT_SECRET")), nil
		})

		if err != nil || !token.Valid {
			handleUnauthorized(c)
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			handleUnauthorized(c)
			return
		}

		// Convert float64 to int64
		userIDFloat, ok := claims["user_id"].(float64)
		if !ok {
			handleUnauthorized(c)
			return
		}
		userID := int64(userIDFloat)

		c.Set("user_id", userID)
		c.Next()
	}
}

func handleUnauthorized(c *gin.Context) {
	// 檢查是否為 API 請求
	if strings.HasPrefix(c.Request.URL.Path, "/vocabulary/") ||
		strings.HasPrefix(c.Request.URL.Path, "/news/") ||
		strings.HasPrefix(c.Request.URL.Path, "/flashcards/") {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
	} else {
		c.Redirect(http.StatusFound, "/login")
	}
	c.Abort()
}
// 檢查使用者是否登入
func IsAuthenticated() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString, err := c.Cookie("token")
		if err != nil {
			c.Set("authenticated", false)
			c.Next()
			return
		}

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return []byte(os.Getenv("JWT_SECRET")), nil
		})

		if err != nil || !token.Valid {
			c.Set("authenticated", false)
			c.Next()
			return
		}

		c.Set("authenticated", true)
		c.Next()
	}
}