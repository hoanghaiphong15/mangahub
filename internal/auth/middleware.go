package auth

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
)

// JWTMiddleware validates the JWT token found in the Authorization header [cite: 820]
func JWTMiddleware(secret []byte) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. Check for the Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is required"})
			c.Abort()
			return
		}

		// 2. Extract the token (Format: "Bearer <token>")
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header format"})
			c.Abort()
			return
		}
		tokenString := parts[1]

		// 3. Parse and validate the token
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			// Ensure the signing method is what we expect (HMAC)
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return secret, nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		// 4. Extract claims and set user_id in the context for handlers to use
		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			userID, ok := claims["user_id"].(string)
			if !ok {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token payload"})
				c.Abort()
				return
			}
			// Store the user_id in the Gin context
			c.Set("user_id", userID)
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Failed to process token claims"})
			c.Abort()
			return
		}

		// 5. Token is valid, proceed to the next handler
		c.Next()
	}
}