package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

type Config struct {
	KeycloakURL   string
	KeycloakRealm string
	RequiredRole  string
	Port          string
}

func LoadConfig() (*Config, error) {
	// Загружаем .env файл из корневой директории проекта
	projectRoot, err := findProjectRoot()
	if err != nil {
		return nil, fmt.Errorf("failed to find project root: %v", err)
	}

	envPath := filepath.Join(projectRoot, ".env")
	err = godotenv.Load(envPath)
	if err != nil {
		// Логируем ошибку, но продолжаем работу - переменные могут быть установлены другим способом
		fmt.Printf("Warning: Error loading .env file: %v\n", err)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8000" // значение по умолчанию
	}

	config := &Config{
		KeycloakURL:   getEnvOrDefault("KEYCLOAK_URL", "http://localhost:8080"),
		KeycloakRealm: getEnvOrDefault("KEYCLOAK_REALM", "reports-realm"),
		RequiredRole:  "prothetic_user",
		Port:          port,
	}

	// Проверяем обязательные переменные
	if config.KeycloakURL == "" || config.KeycloakRealm == "" {
		return nil, fmt.Errorf("required environment variables are not set")
	}

	return config, nil
}

// getEnvOrDefault возвращает значение переменной окружения или значение по умолчанию
func getEnvOrDefault(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// findProjectRoot ищет корневую директорию проекта
func findProjectRoot() (string, error) {
	// Начинаем с текущей директории
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// Ищем go.mod файл, поднимаясь по дереву директорий
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("could not find project root (no go.mod file found)")
		}
		dir = parent
	}
}
