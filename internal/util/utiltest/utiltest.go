package utiltest

import (
	"os"
	"path/filepath"
)

// GetProjectRoot returns the root path of the project.
func GetProjectRoot() (string, error) {
	// Get the current working directory
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// Iterate up the directory tree to find the project root
	for {
		// Check if a specific file exists (like go.mod or a main.go)
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil // Return the directory if go.mod is found
		}
		if _, err := os.Stat(filepath.Join(dir, "main.go")); err == nil {
			return dir, nil // Return the directory if main.go is found
		}
		// Move up one directory
		parentDir := filepath.Dir(dir)
		if parentDir == dir { // Stop if we've reached the root
			break
		}
		dir = parentDir
	}
	return "", os.ErrNotExist // Return an error if the project root wasn't found
}
