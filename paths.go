package main

import (
	"fmt"
	"os"
	"path/filepath"
)

// executableDir returns the directory containing the running executable.
func executableDir() (string, error) {
	execPath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("resolving executable path: %w", err)
	}

	return filepath.Dir(execPath), nil
}
