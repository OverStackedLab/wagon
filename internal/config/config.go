package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// Dir returns the Wagon config directory without creating it.
// WAGON_CONFIG_DIR overrides everything, then XDG_CONFIG_HOME, then ~/.config.
func Dir() (string, error) {
	if dir := os.Getenv("WAGON_CONFIG_DIR"); dir != "" {
		return dir, nil
	}
	if base := os.Getenv("XDG_CONFIG_HOME"); base != "" {
		return filepath.Join(base, "wagon"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	return filepath.Join(home, ".config", "wagon"), nil
}

// JobsPath returns the saved jobs file path.
func JobsPath() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "jobs.yaml"), nil
}

// RunsPath returns the job run history file path.
func RunsPath() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "job-runs.yaml"), nil
}

// LocksDir returns the directory that holds per-job lock files.
func LocksDir() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "locks"), nil
}

// LogsDir returns where per-job logs are written: ~/Library/Logs/wagon on
// macOS, otherwise a logs folder under the config directory.
func LogsDir() (string, error) {
	if dir := os.Getenv("WAGON_CONFIG_DIR"); dir != "" {
		return filepath.Join(dir, "logs"), nil
	}
	if runtime.GOOS == "darwin" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory: %w", err)
		}
		return filepath.Join(home, "Library", "Logs", "wagon"), nil
	}
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "logs"), nil
}
