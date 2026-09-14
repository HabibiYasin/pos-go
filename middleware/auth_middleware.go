package middleware

import (
	"pos-go/utils"
	"strings"

	"github.com/gin-gonic/gin"
)

// Explicit Authorization takes precedence over legacy shared cookies, even when invalid.
func getTokenFromRequest(c *gin.Context) string {
	if auth := c.GetHeader("Authorization"); auth != "" {
		parts := strings.Fields(auth)
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			return parts[1]
		}
		return ""
	}
	// Compatibility for clients that still use cookie authentication.
	if token, err := c.Cookie("token"); err == nil && token != "" {
		return token
	}
	return ""
}

// AuthMiddleware untuk memvalidasi JWT token dari cookie atau Authorization header
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := getTokenFromRequest(c)
		if token == "" {
			utils.ErrorResponseUnauthorized(c, "Token tidak ditemukan. Kirim via cookie 'token' atau header Authorization: Bearer <token>")
			c.Abort()
			return
		}

		// Validasi token
		claims, err := utils.ValidateToken(token)
		if err != nil {
			utils.ErrorResponseUnauthorized(c, "Token tidak valid")
			c.Abort()
			return
		}

		// Simpan claims ke context (bisa dipakai di handler)
		c.Set("user_id", claims.UserID)
		c.Set("email", claims.Email)
		c.Set("role", claims.Role)

		c.Next()
	}
}
