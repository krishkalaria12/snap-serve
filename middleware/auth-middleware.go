package middleware

import (
	"log"
	"strconv"

	"github.com/go-pkgz/auth/v2/token"
	"github.com/gofiber/fiber/v2"
	"github.com/krishkalaria12/snap-serve/auth"
)

func AuthMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		var tokenStr string

		if authHeader != "" && len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			tokenStr = authHeader[7:]
		} else {
			tokenStr = c.Cookies("JWT")
		}

		if tokenStr == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"status":  "error",
				"message": "You are not authorized!",
				"data":    nil,
			})
		}

		// Validate token using go-pkgz/auth
		claims, err := auth.GetAuthService().TokenService().Parse(tokenStr)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"message": "Invalid token",
				"status":  "error",
				"data":    nil,
			})
		}

		// Store user and claims in context
		c.Locals("user", *claims.User)
		c.Locals("claims", claims)

		return c.Next()
	}
}

func CheckUserLoggedIn(c *fiber.Ctx) (uint, error) {
	user := c.Locals("user").(token.User)

	userID, err := strconv.ParseUint(user.ID, 10, 32)
	if err != nil {
		log.Printf("Failed to parse user ID: %s", user.ID)
	}

	return uint(userID), err
}
