package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

func Config(envVar string) string {
	err := godotenv.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading .env file: %v\n", err)
		panic(err)
	}

	envVarValue := os.Getenv(envVar)
	if envVarValue == "" {
		fmt.Fprintf(os.Stderr, "%s not set\n", envVar)
		os.Exit(1)
	}

	return envVarValue
}
