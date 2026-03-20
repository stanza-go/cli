package main

import (
	"fmt"
	"os"
	"path/filepath"
)

const defaultDirName = ".stanza"

// resolveDataDir returns the absolute path to the Stanza data directory.
// Priority: explicit flag > DATA_DIR env > ~/.stanza/
func resolveDataDir(flagValue string) (string, error) {
	if flagValue != "" {
		return filepath.Abs(flagValue)
	}

	if env := os.Getenv("DATA_DIR"); env != "" {
		return filepath.Abs(env)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}

	return filepath.Join(home, defaultDirName), nil
}
