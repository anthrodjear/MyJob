package middleware

import (
	"errors"
	"strings"

	"github.com/gin-gonic/gin"

	"backend/internal/auth"
	"backend/internal/httpresp"
)

// AuthMiddleware returns a Gin handler that validates JWT tokens.
// It extracts the token from the Authorization header, validates it,
// and stores the claims in the request context for downstream handlers.
func AuthMiddleware(authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			httpresp.Unauthorized(c, "MISSING_TOKEN", "authorization header required")
			c.Abort()
			return
		}

		// Expect "Bearer <token>"
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			httpresp.Unauthorized(c, "INVALID_TOKEN_FORMAT", "authorization header must be 'Bearer <token>'")
			c.Abort()
			return
		}

		tokenString := parts[1]
		claims, err := authService.ValidateTokenWithSession(c.Request.Context(), tokenString)
		if err != nil {
			if errors.Is(err, auth.ErrTokenExpired) {
				httpresp.Unauthorized(c, "TOKEN_EXPIRED", "token has expired")
				c.Abort()
				return
			}
			if errors.Is(err, auth.ErrSessionInvalidated) {
				httpresp.Unauthorized(c, "SESSION_INVALIDATED", "session invalidated, please log in again")
				c.Abort()
				return
			}
			httpresp.Unauthorized(c, "INVALID_TOKEN", "invalid token")
			c.Abort()
			return
		}

		// Store claims in context for downstream handlers
		c.Set("auth_claims", claims)
		c.Set("user_id", claims.UserID)

		c.Next()
	}
}

// GetClaims retrieves the JWT claims from the request context.
// Returns nil if not set (e.g., middleware not applied).
func GetClaims(c *gin.Context) *auth.Claims {
	val, ok := c.Get("auth_claims")
	if !ok {
		return nil
	}
	claims, ok := val.(*auth.Claims)
	if !ok {
		return nil
	}
	return claims
}

// GetUserID retrieves the user ID from the request context.
// Returns empty string if not set.
func GetUserID(c *gin.Context) string {
	val, ok := c.Get("user_id")
	if !ok {
		return ""
	}
	id, ok := val.(string)
	if !ok {
		return ""
	}
	return id
}
