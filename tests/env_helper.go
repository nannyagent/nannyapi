package tests

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// LoadEnv loads environment variables from .env file in the project root
// It searches for .env starting from the current directory and going up
func LoadEnv(t *testing.T) {
	dir, err := os.Getwd()
	if err != nil {
		t.Logf("Failed to get current directory: %v", err)
		return
	}

	var envPath string
	for {
		path := filepath.Join(dir, ".env")
		if _, err := os.Stat(path); err == nil {
			envPath = path
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	if envPath == "" {
		t.Log("No .env file found")
		return
	}

	file, err := os.Open(envPath)
	if err != nil {
		t.Logf("Failed to open .env file: %v", err)
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Remove quotes if present
		if len(value) >= 2 && ((value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'')) {
			value = value[1 : len(value)-1]
		}

		// Only set if not already set (allow override from shell)
		if os.Getenv(key) == "" {
			os.Setenv(key, value)
		}
	}

	if err := scanner.Err(); err != nil {
		t.Logf("Error reading .env file: %v", err)
	}
}
