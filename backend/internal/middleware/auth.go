package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v4"
)

type Claims struct {
	UserID  string `json:"user_id"`
	IsAdmin bool   `json:"is_admin"`
	jwt.RegisteredClaims
}

// JWT validates Bearer tokens and injects claims into context.
func JWT(secret string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		auth := c.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			return fiber.NewError(fiber.StatusUnauthorized, "missing token")
		}
		tokenStr := strings.TrimPrefix(auth, "Bearer ")

		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fiber.NewError(fiber.StatusUnauthorized, "invalid signing method")
			}
			return []byte(secret), nil
		})
		if err != nil || !token.Valid {
			return fiber.NewError(fiber.StatusUnauthorized, "invalid token")
		}

		c.Locals("user_id", claims.UserID)
		c.Locals("is_admin", claims.IsAdmin)
		return c.Next()
	}
}

// AdminOnly rejects non-admin users.
func AdminOnly() fiber.Handler {
	return func(c *fiber.Ctx) error {
		isAdmin, _ := c.Locals("is_admin").(bool)
		if !isAdmin {
			return fiber.NewError(fiber.StatusForbidden, "admin only")
		}
		return c.Next()
	}
}

// UserID extracts the authenticated user ID from context.
func UserID(c *fiber.Ctx) string {
	id, _ := c.Locals("user_id").(string)
	return id
}
