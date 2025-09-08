package models

import (
	"gorm.io/gorm"
)

type Image struct {
	gorm.Model
	UserID       uint   `json:"user_id" gorm:"not null;index"`
	Filename     string `json:"filename" gorm:"not null"`
	OriginalURL  string `json:"original_url" gorm:"not null"`
	ProcessedURL string `json:"processed_url,omitempty"`
	Status       string `json:"status" gorm:"not null;default:'pending'"`

	// Relationship
	User User `gorm:"foreignKey:UserID" json:"user"`
}
