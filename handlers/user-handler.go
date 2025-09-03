package handler

import (
	"errors"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/krishkalaria12/snap-serve/database"
	"github.com/krishkalaria12/snap-serve/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
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

func UpdateUser(c *fiber.Ctx) error {
	type UpdateUser struct {
		Username string `json:"username"`
		FullName string `json:"name"`
	}
	type UserResponse struct {
		ID       uint   `json:"id"`
		Email    string `json:"email"`
		Username string `json:"username"`
		FullName string `json:"name"`
	}

	var userInput UpdateUser
	if err := c.BodyParser(&userInput); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Invalid input",
			"status":  "error",
			"data":    nil, // Don't expose internal errors
		})
	}

	id := c.Params("id")

	// Validate ID parameter
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "User ID is required",
			"status":  "error",
			"data":    nil,
		})
	}

	db := database.GetDB()
	var user models.User

	if err := db.First(&user, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"message": "User not found",
				"status":  "error",
				"data":    nil,
			})
		}
		// Handle other database errors
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Database error",
			"status":  "error",
			"data":    nil,
		})
	}

	// Optional: Add validation for input fields
	if userInput.Username == "" || userInput.FullName == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Username and name are required",
			"status":  "error",
			"data":    nil,
		})
	}

	// Optional: Check if username already exists (if username should be unique)
	var existingUser models.User
	if err := db.Where("username = ? AND id != ?", userInput.Username, id).First(&existingUser).Error; err == nil {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"message": "Username already taken",
			"status":  "error",
			"data":    nil,
		})
	}

	// Update user fields
	user.Username = userInput.Username
	user.FullName = userInput.FullName

	// Save changes and handle errors
	if err := db.Save(&user).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Failed to update user",
			"status":  "error",
			"data":    nil,
		})
	}

	response := UserResponse{
		ID:       user.ID,
		Email:    user.Email,
		Username: user.Username,
		FullName: user.FullName,
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status":  "success",
		"message": "User successfully updated",
		"data":    response,
	})
}

func DeleteUser(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "User ID is required",
			"status":  "error",
			"data":    nil,
		})
	}

	db := database.GetDB()
	var user models.User

	if err := db.First(&user, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"message": "User not found",
				"status":  "error",
				"data":    nil,
			})
		}
		// Handle other database errors
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Database error",
			"status":  "error",
			"data":    nil,
		})
	}

	// Delete the user and handle errors properly
	if err := db.Delete(&user).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Failed to delete user",
			"status":  "error",
			"data":    nil,
		})
	}

	// Clear JWT cookie (fix the expiry time)
	c.Cookie(&fiber.Cookie{
		Name:     "JWT",
		Value:    "",
		Expires:  time.Now().Add(-time.Hour), // Past time to clear cookie
		HTTPOnly: true,
		Secure:   false,
		SameSite: "Lax",
	})

	return c.JSON(fiber.Map{
		"status":  "success",
		"message": "User deleted successfully",
		"data":    nil,
	})
}
