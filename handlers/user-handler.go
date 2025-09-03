package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/krishkalaria12/snap-serve/database"
	"github.com/krishkalaria12/snap-serve/models"
	"golang.org/x/crypto/bcrypt"
)

func hashPassword(password string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), 10)
	return string(hashed), err
}

func GetUser(c *fiber.Ctx) error {
	type UserResponse struct {
		Email    string `json:"email"`
		Username string `json:"username"`
		FullName string `json:"name"`
	}

	id := c.Params("id")

	db := database.GetDB()
	user := models.User{}
	db.Find(&user, id)

	if user.Username == "" {
		return c.Status(404).JSON(fiber.Map{"status": "error", "message": "No user found with ID", "data": nil})
	}

	userResponse := UserResponse{
		Email:    user.Email,
		Username: user.Username,
		FullName: user.FullName,
	}

	return c.JSON(fiber.Map{"status": "success", "message": "User found", "data": userResponse})
}

func CreateUser(c *fiber.Ctx) error {
	type NewUser struct {
		Email    string `json:"email"`
		Username string `json:"username"`
		FullName string `json:"name"`
	}

	db := database.GetDB()
	user := new(models.User)

	if err := c.BodyParser(user); err != nil {
		return c.Status(500).JSON(fiber.Map{"status": "error", "message": "Wrong Input Data Format", "data": err})
	}

	hash, err := hashPassword(user.Password)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"status": "error", "message": "Failed to hash password", "data": err})
	}
	user.Password = hash

	if err := db.Create(&user); err != nil {
		return c.Status(500).JSON(fiber.Map{"status": "error", "message": "Failed to create user", "data": err})
	}

	newuser := NewUser{
		Email:    user.Email,
		Username: user.Username,
		FullName: user.FullName,
	}

	return c.Status(200).JSON(fiber.Map{"status": "success", "message": "User created successfully", "data": newuser})
}
