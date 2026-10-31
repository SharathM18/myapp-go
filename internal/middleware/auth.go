package middleware

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/sharathm18/myapp/internal/server"
)

type AuthMiddleware struct {
	server *server.APIServer
}

func NewAuthMiddleware(s *server.APIServer) *AuthMiddleware {
	return &AuthMiddleware{server: s}
}

// RequireAuthentication validates the JWT and injects user context
func (m *AuthMiddleware) RequireAuthentication() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
				return echo.NewHTTPError(http.StatusUnauthorized, "Missing or invalid authorization header")
			}

			tokenString := strings.TrimPrefix(authHeader, "Bearer ")

			// TODO: Replace with actual JWT parsing and validation (e.g., using golang-jwt/jwt)
			// claims, err := parseAndValidateJWT(tokenString, m.server.Config.JWTSecret)
			if tokenString == "expired-token" {
				return echo.NewHTTPError(http.StatusUnauthorized, "Token has expired")
			}

			// Mock data extracted from the validated JWT claims
			userID := "usr_12345"
			role := "admin"

			// Store in Echo context.
			// Your Enhancers and Handlers will read exactly these keys.
			c.Set("user_id", userID)
			c.Set("user_role", role)

			return next(c)
		}
	}
}
