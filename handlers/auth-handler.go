package handler

import (
	"errors"
	"strconv"
	"time"

	"github.com/go-pkgz/auth/v2/token"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/krishkalaria12/snap-serve/auth"
	"github.com/krishkalaria12/snap-serve/database"
	"github.com/krishkalaria12/snap-serve/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"net/mail"
)

func isEmail(identity string) bool {
	_, err := mail.ParseAddress(identity)
	return err == nil
}

func getUserByEmail(email string) (*models.User, error) {
	db := database.GetDB()
	var user models.User
	if err := db.Where(&models.User{Email: email}).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func getUserByUsername(username string) (*models.User, error) {
	db := database.GetDB()
	var user models.User
	if err := db.Where(&models.User{Username: username}).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func checkPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}


// Custom login handler that integrates with go-pkgz/auth
func Login(c *fiber.Ctx) error {
	type LoginData struct {
		Identity string `json:"identity"`
		Password string `json:"password"`
	}

	type UserResponse struct {
		ID       uint   `json:"id"`
		Email    string `json:"email"`
		Username string `json:"username"`
		FullName string `json:"name"`
		Token    string `json:"token"`
	}

	input := new(LoginData)
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Invalid request body",
			"status":  "error",
			"data":    nil,
		})
	}

	// Validate credentials using auth service  
	valid, err := auth.ValidateUserCredentials(input.Identity, input.Password)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Database error",
			"status":  "error",
			"data":    nil,
		})
	}

	if !valid {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"message": "Invalid identity or password",
			"status":  "error",
			"data":    nil,
		})
	}

	// Get user model for response
	var userModel *models.User
	if isEmail(input.Identity) {
		userModel, err = getUserByEmail(input.Identity)
	} else {
		userModel, err = getUserByUsername(input.Identity)
	}

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Database error",
			"status":  "error",
			"data":    nil,
		})
	}

	if userModel == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"message": "Invalid identity or password",
			"status":  "error",
			"data":    nil,
		})
	}

	if !checkPasswordHash(input.Password, userModel.Password) {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"status":  "error",
			"message": "Invalid identity or password",
			"data":    nil,
		})
	}

	// Create JWT token using go-pkgz/auth
	user := token.User{
		ID:    strconv.FormatUint(uint64(userModel.ID), 10), // Convert to string
		Name:  userModel.FullName,
		Email: userModel.Email,
		Attributes: map[string]interface{}{
			"email":    userModel.Email,
			"username": userModel.Username,
			"user_id":  userModel.ID,
		},
	}

	claims := token.Claims{
		User: &user,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    auth.GetAuthService().TokenService().Issuer,
			Audience:  []string{"snap-serve-app"},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 24)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	// Generate JWT token
	tokenStr, err := auth.GetAuthService().TokenService().Token(claims)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Failed to generate token",
			"status":  "error",
			"data":    nil,
		})
	}

	// Set JWT cookie (optional, for web clients)
	c.Cookie(&fiber.Cookie{
		Name:     "JWT",
		Value:    tokenStr,
		Expires:  time.Now().Add(time.Hour * 24 * 7),
		HTTPOnly: true,
		Secure:   false, // Set to true in production with HTTPS
		SameSite: "Lax",
	})

	// Return response with token
	response := UserResponse{
		ID:       userModel.ID,
		Email:    userModel.Email,
		Username: userModel.Username,
		FullName: userModel.FullName,
		Token:    tokenStr,
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Login successful",
		"status":  "success",
		"data":    response,
	})
}

func Logout(c *fiber.Ctx) error {
	// Clear JWT cookie
	c.Cookie(&fiber.Cookie{
		Name:     "JWT",
		Value:    "",
		Expires:  time.Now().Add(-time.Hour),
		HTTPOnly: true,
	})

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Logout successful",
		"status":  "success",
		"data":    nil,
	})
}
