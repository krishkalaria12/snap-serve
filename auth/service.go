package auth

import (
	"errors"
	"net/mail"
	"time"

	"github.com/go-pkgz/auth/v2"
	"github.com/go-pkgz/auth/v2/avatar"
	"github.com/go-pkgz/auth/v2/provider"
	"github.com/go-pkgz/auth/v2/token"
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
		return ValidateUserCredentials(identity, password)
	}))

	authService = service
	return service
}

// Get the auth service instance
func GetAuthService() *auth.Service {
	return authService
}

// ValidateUserCredentials validates user credentials against your database
func ValidateUserCredentials(identity, password string) (bool, error) {
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