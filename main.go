package main

import (
	"fmt"
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/krishkalaria12/snap-serve/database"
	"github.com/krishkalaria12/snap-serve/models"
)

func main() {
	_ = database.GetDB()

	// Run migrations
	err := database.MigrateModels(&models.User{})
	if err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	app := fiber.New()

	app.Get("/hello", func(c *fiber.Ctx) error {
		return c.SendString("Hello, World!")
	})

	// close the database connection
	defer func() {
		if err := database.CloseDB(); err != nil {
			fmt.Printf("Enter closing the Database connection %v", err)
			log.Fatal(err)
		}
	}()

	fmt.Println("Server is listening at the port 3000")
	log.Fatal(app.Listen(":3000"))
}
