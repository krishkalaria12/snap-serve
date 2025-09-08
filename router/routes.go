package router

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	handler "github.com/krishkalaria12/snap-serve/handlers"
	"github.com/krishkalaria12/snap-serve/middleware"
)

func SetupRoutes(app *fiber.App) {
	api := app.Group("/api", logger.New())
	api.Get("/hello", handler.Hello)

	// Auth
	auth := api.Group("/auth")
	auth.Post("/login", handler.Login)

	// User
	user := api.Group("/user")
	user.Get("/:id", handler.GetUser)
	user.Post("/", handler.CreateUser)
	user.Put("/:id", middleware.AuthMiddleware(), handler.UpdateUser)
	user.Delete("/:id", middleware.AuthMiddleware(), handler.DeleteUser)

	image := api.Group("/image")
	image.Post("/upload", middleware.AuthMiddleware(), handler.UploadImage)
}
