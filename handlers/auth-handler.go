package handler

import (
	"errors"
	"net/mail"
	"time"

	"github.com/go-pkgz/auth/v2"
	"github.com/go-pkgz/auth/v2/avatar"
	"github.com/go-pkgz/auth/v2/provider"
	"github.com/go-pkgz/auth/v2/token"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/krishkalaria12/snap-serve/config"
	"github.com/krishkalaria12/snap-serve/database"
	"github.com/krishkalaria12/snap-serve/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// Global auth service instance
var authService *auth.Service

// Initialize auth service
func SetupAuthService() *auth.Service {
	options := auth.Opts{
		SecretReader: token.SecretFunc(func(id string) (string, error) {
			// Use environment variable in production
			return config.Config("JWT_SECRET"), nil
		}),
		TokenDuration:  time.Hour * 24,     // JWT token duration
		CookieDuration: time.Hour * 24 * 7, // Cookie duration
		Issuer:         "snap-serve-app",
		URL:            "http://localhost:3000", // Your app URL
		AvatarStore:    avatar.NewLocalFS("/tmp/avatars"),
	}

	// Create auth service
	service := auth.NewService(options)

	// Add direct authentication provider that uses your database
	service.AddDirectProvider("local", provider.CredCheckerFunc(func(identity, password string) (bool, error) {
		return validateUserCredentials(identity, password)
	}))

	authService = service
	return service
}

// Get the auth service instance
func GetAuthService() *auth.Service {
	return authService
}

// Validate user credentials against your database
func validateUserCredentials(identity, password string) (bool, error) {
	var user *models.User
	var err error

	if isEmail(identity) {
		user, err = getUserByEmail(identity)
	} else {
		user, err = getUserByUsername(identity)
	}

	if err != nil {
		return false, err
	}

	if user == nil {
		return false, nil // User not found
	}

	// Check password
	if !checkPasswordHash(password, user.Password) {
		return false, nil // Invalid password
	}

	return true, nil
}

func checkPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func isEmail(identity string) bool {
	_, err := mail.ParseAddress(identity)
	return err == nil
}

func getUserByEmail(email string) (*models.User, error) {
	db := database.GetDB()
	var user models.User
	if err := db.Where("email = ?", email).First(&user).Error; err != nil {
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
	if err := db.Where("username = ?", username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
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

	// Validate credentials
	var userModel *models.User
	var err error

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
		ID:    "user_" + string(rune(userModel.ID)), // Convert to string
		Name:  userModel.FullName,
		Email: userModel.Email,
		Attributes: map[string]interface{}{
			"username": userModel.Username,
		},
	}

	claims := token.Claims{
		User: &user,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    authService.TokenService().Issuer,
			Audience:  []string{"snap-serve-app"},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 24)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	// Generate JWT token
	tokenStr, err := authService.TokenService().Token(claims)
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
