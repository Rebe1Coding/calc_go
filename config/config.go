package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Username    string
	Password    string
	DeepSeekURL string
}

func Load() *Config {
	// Загрузка .env файла
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	return &Config{
		Username:    getEnv("USERNAME", "user"),
		Password:    getEnv("PASSWORD", "123"),
		DeepSeekURL: getEnv("DEEPSEEK_URL", "URL not set"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	boolValue, err := strconv.ParseBool(value)
	if err != nil {
		return defaultValue
	}

	return boolValue
}
