package middleware

import (
	"github.com/esdrassantos06/go-shortener/internal/core/auth"
	"github.com/gofiber/fiber/v3"
)

type AuthMiddleware struct {
	validator *auth.SessionValidator
}

func NewAuthMiddleware(validator *auth.SessionValidator) *AuthMiddleware {
	return &AuthMiddleware{validator: validator}
}

func (am *AuthMiddleware) RequireAuth(c fiber.Ctx) error {
	cookieHeader := c.Get("Cookie")

	sessionToken, err := auth.GetSessionFromCookie(cookieHeader)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{
			"error": "Unauthorized: No session found",
		})
	}

	userID, err := am.validator.ValidateSession(c.Context(), sessionToken)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{
			"error": "Unauthorized: Invalid or expired session",
		})
	}

	c.Locals("userID", userID)

	return c.Next()
}
