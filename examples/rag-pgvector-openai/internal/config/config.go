package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the production RAG system
type Config struct {
	Database DatabaseConfig
	OpenAI   OpenAIConfig
	Vector   VectorConfig
	App      AppConfig
}

type DatabaseConfig struct {
	Host           string
	Port           int
	Name           string
	User           string
	Password       string
	SSLMode        string
	MaxConnections int
	MinConnections int
}

type OpenAIConfig struct {
	APIKey         string
	Model          string
	EmbeddingModel string
	MaxTokens      int
	Temperature    float64
	RPMLimit       int
	TPMLimit       int
}

type VectorConfig struct {
	Dimensions          int
	MaxSearchResults    int
	SimilarityThreshold float64
}

type AppConfig struct {
	LogLevel       string
	ServerPort     int
	RequestTimeout time.Duration
}

// LoadConfig loads configuration from environment variables
func LoadConfig() (*Config, error) {
	// Try to load .env file (ignore error if it doesn't exist)
	_ = godotenv.Load()

	cfg := &Config{
		Database: DatabaseConfig{
			Host:           getEnvWithDefault("DB_HOST", "localhost"),
			Port:           getEnvAsInt("DB_PORT", 5432),
			Name:           getEnvWithDefault("DB_NAME", "rag_production"),
			User:           getEnvWithDefault("DB_USER", "postgres"),
			Password:       getEnvWithDefault("DB_PASSWORD", ""),
			SSLMode:        getEnvWithDefault("DB_SSL_MODE", "disable"),
			MaxConnections: getEnvAsInt("DB_MAX_CONNECTIONS", 25),
			MinConnections: getEnvAsInt("DB_MIN_CONNECTIONS", 5),
		},
		OpenAI: OpenAIConfig{
			APIKey:         getEnvWithDefault("OPENAI_API_KEY", ""),
			Model:          getEnvWithDefault("OPENAI_MODEL", "gpt-4"),
			EmbeddingModel: getEnvWithDefault("OPENAI_EMBEDDING_MODEL", "text-embedding-ada-002"),
			MaxTokens:      getEnvAsInt("OPENAI_MAX_TOKENS", 1500),
			Temperature:    getEnvAsFloat("OPENAI_TEMPERATURE", 0.7),
			RPMLimit:       getEnvAsInt("OPENAI_RPM_LIMIT", 3000),
			TPMLimit:       getEnvAsInt("OPENAI_TPM_LIMIT", 90000),
		},
		Vector: VectorConfig{
			Dimensions:          getEnvAsInt("VECTOR_DIMENSIONS", 1536),
			MaxSearchResults:    getEnvAsInt("MAX_SEARCH_RESULTS", 10),
			SimilarityThreshold: getEnvAsFloat("SIMILARITY_THRESHOLD", 0.7),
		},
		App: AppConfig{
			LogLevel:       getEnvWithDefault("LOG_LEVEL", "info"),
			ServerPort:     getEnvAsInt("SERVER_PORT", 8080),
			RequestTimeout: getEnvAsDuration("REQUEST_TIMEOUT", 30*time.Second),
		},
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.OpenAI.APIKey == "" {
		return fmt.Errorf("OPENAI_API_KEY is required")
	}

	if c.Database.Password == "" {
		return fmt.Errorf("DB_PASSWORD is required")
	}

	if c.Vector.Dimensions <= 0 {
		return fmt.Errorf("VECTOR_DIMENSIONS must be positive")
	}

	if c.Vector.SimilarityThreshold < 0 || c.Vector.SimilarityThreshold > 1 {
		return fmt.Errorf("SIMILARITY_THRESHOLD must be between 0 and 1")
	}

	return nil
}

// GetDatabaseURL returns the PostgreSQL connection string
func (c *Config) GetDatabaseURL() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Database.Host,
		c.Database.Port,
		c.Database.User,
		c.Database.Password,
		c.Database.Name,
		c.Database.SSLMode,
	)
}

// Helper functions for environment variable parsing

func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if valueStr := os.Getenv(key); valueStr != "" {
		if value, err := strconv.Atoi(valueStr); err == nil {
			return value
		}
	}
	return defaultValue
}

func getEnvAsFloat(key string, defaultValue float64) float64 {
	if valueStr := os.Getenv(key); valueStr != "" {
		if value, err := strconv.ParseFloat(valueStr, 64); err == nil {
			return value
		}
	}
	return defaultValue
}

func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	if valueStr := os.Getenv(key); valueStr != "" {
		if value, err := time.ParseDuration(valueStr); err == nil {
			return value
		}
	}
	return defaultValue
}
