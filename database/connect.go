package database

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/krishkalaria12/snap-serve/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	instance *gorm.DB
	once     sync.Once
)

func GetDB() *gorm.DB {
	once.Do(func() {
		instance = connectDB()
	})

	return instance
}

func connectDB() *gorm.DB {
	dsn := config.Config("DATABASE_URL")

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
		panic(err)
	}

	// Get the underlying SQL DB object for connection pooling
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("Failed to get DB object: %v", err)
	}

	// Configure connection pool
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetMaxOpenConns(10)
	sqlDB.SetConnMaxLifetime(time.Hour)
	sqlDB.SetConnMaxIdleTime(30 * time.Minute)

	// Verify connection
	if err := sqlDB.Ping(); err != nil {
		log.Fatalf("Failed to ping DB: %v", err)
	}

	fmt.Println("Successfully connected to Neon Postgres database!")
	return db
}

// MigrateModels runs auto migration for your models
func MigrateModels(models ...interface{}) error {
	db := GetDB()
	return db.AutoMigrate(models...)
}

func CloseDB() error {
	if instance != nil {
		sqlDB, err := instance.DB()
		if err != nil {
			return err
		}

		return sqlDB.Close()
	}

	return nil
}
