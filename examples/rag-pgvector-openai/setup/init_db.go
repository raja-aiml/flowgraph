package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found: %v", err)
	}

	// Get database connection parameters
	host := getEnvWithDefault("DB_HOST", "localhost")
	port := getEnvWithDefault("DB_PORT", "5432")
	user := getEnvWithDefault("DB_USER", "postgres")
	password := os.Getenv("DB_PASSWORD")
	dbname := getEnvWithDefault("DB_NAME", "rag_production")
	sslmode := getEnvWithDefault("DB_SSL_MODE", "disable")

	if password == "" {
		log.Fatal("DB_PASSWORD environment variable is required")
	}

	// Connect to PostgreSQL (without specifying database)
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s sslmode=%s",
		host, port, user, password, sslmode)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	defer db.Close()

	// Test connection
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping PostgreSQL: %v", err)
	}

	fmt.Println("Connected to PostgreSQL successfully!")

	// Create database if it doesn't exist
	if err := createDatabaseIfNotExists(db, dbname); err != nil {
		log.Fatalf("Failed to create database: %v", err)
	}

	// Connect to the specific database
	dbConnStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, password, dbname, sslmode)

	appDB, err := sql.Open("postgres", dbConnStr)
	if err != nil {
		log.Fatalf("Failed to connect to application database: %v", err)
	}
	defer appDB.Close()

	// Enable pgvector extension
	if err := enablePgVector(appDB); err != nil {
		log.Fatalf("Failed to enable pgvector: %v", err)
	}

	fmt.Printf("Database '%s' is ready for production RAG!\n", dbname)
	fmt.Println("✅ Setup complete!")
}

func createDatabaseIfNotExists(db *sql.DB, dbname string) error {
	// Check if database exists
	var exists bool
	query := "SELECT EXISTS(SELECT datname FROM pg_catalog.pg_database WHERE datname = $1);"
	err := db.QueryRow(query, dbname).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check if database exists: %w", err)
	}

	if exists {
		fmt.Printf("Database '%s' already exists\n", dbname)
		return nil
	}

	// Create database
	createQuery := fmt.Sprintf("CREATE DATABASE %s;", dbname)
	_, err = db.Exec(createQuery)
	if err != nil {
		return fmt.Errorf("failed to create database: %w", err)
	}

	fmt.Printf("Created database '%s'\n", dbname)
	return nil
}

func enablePgVector(db *sql.DB) error {
	// Enable pgvector extension
	query := "CREATE EXTENSION IF NOT EXISTS vector;"
	_, err := db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to enable vector extension: %w", err)
	}

	fmt.Println("Enabled pgvector extension")

	// Verify extension is installed
	var extExists bool
	checkQuery := "SELECT EXISTS(SELECT 1 FROM pg_extension WHERE extname = 'vector');"
	err = db.QueryRow(checkQuery).Scan(&extExists)
	if err != nil {
		return fmt.Errorf("failed to verify vector extension: %w", err)
	}

	if !extExists {
		return fmt.Errorf("pgvector extension is not properly installed")
	}

	fmt.Println("✅ pgvector extension verified")
	return nil
}

func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
