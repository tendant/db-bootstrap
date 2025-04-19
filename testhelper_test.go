package dbstrap

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

// setupTestDatabase ensures a PostgreSQL container is running and returns the connection URL
func setupTestDatabase(t *testing.T) (string, func()) {
	// Check if we already have a DATABASE_URL
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL != "" {
		return dbURL, func() {}
	}

	// Start PostgreSQL container using docker-compose
	cmd := exec.Command("docker-compose", "up", "-d")
	err := cmd.Run()
	if err != nil {
		t.Fatalf("Failed to start PostgreSQL container: %v", err)
	}

	// Wait for PostgreSQL to be ready
	dbURL = "postgres://postgres:pwd@localhost:5432/postgres"
	maxRetries := 30
	for i := 0; i < maxRetries; i++ {
		conn, err := pgx.Connect(context.Background(), dbURL)
		if err == nil {
			conn.Close(context.Background())
			break
		}
		if i == maxRetries-1 {
			t.Fatalf("PostgreSQL failed to start after %d attempts: %v", maxRetries, err)
		}
		time.Sleep(1 * time.Second)
		fmt.Printf("Waiting for PostgreSQL to start (attempt %d/%d)...\n", i+1, maxRetries)
	}

	// Return cleanup function
	cleanup := func() {
		cmd := exec.Command("docker-compose", "down")
		cmd.Run()
	}

	return dbURL, cleanup
}

// TestMain is used to set up the testing environment
func TestMain(m *testing.M) {
	// Run tests
	code := m.Run()

	// Exit with the same code
	os.Exit(code)
}
