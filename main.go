package main

import (
	"fmt"
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/krishkalaria12/snap-serve/database"
	handler "github.com/krishkalaria12/snap-serve/handlers"
	"github.com/krishkalaria12/snap-serve/models"
	"github.com/krishkalaria12/snap-serve/router"
)

func main() {
	_ = database.GetDB()

	// Run migrations
	err := database.MigrateModels(&models.User{})
	if err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	app := fiber.New()
	app.Use(cors.New())

	// Initialize auth service
	handler.SetupAuthService()

	// close the database connection
	defer func() {
		if err := database.CloseDB(); err != nil {
			fmt.Printf("Enter closing the Database connection %v", err)
			log.Fatal(err)
		}
	}()

	router.SetupRoutes(app)
	log.Fatal(app.Listen(":3000"))
}
