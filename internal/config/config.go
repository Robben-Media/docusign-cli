package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const AppName = "placeholder-cli"

var (
	ErrConfigDir = errors.New("config directory error")
)

// ConfigDir returns the platform-specific config directory.
// macOS: ~/Library/Application Support/{cli}/
// Linux: $XDG_CONFIG_HOME/{cli}/ (default: ~/.config/{cli}/)
// Windows: %APPDATA%\{cli}\
func ConfigDir() (string, error) {
	var baseDir string

	switch runtime.GOOS {
	case "darwin":
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("%w: get home directory: %w", ErrConfigDir, err)
		}
		baseDir = filepath.Join(homeDir, "Library", "Application Support", AppName)
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			return "", fmt.Errorf("%w: APPDATA not set", ErrConfigDir)
		}
		baseDir = filepath.Join(appData, AppName)
	default: // Linux and other Unix-like systems
		configHome := os.Getenv("XDG_CONFIG_HOME")
		if configHome == "" {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return "", fmt.Errorf("%w: get home directory: %w", ErrConfigDir, err)
			}
			configHome = filepath.Join(homeDir, ".config")
		}
		baseDir = filepath.Join(configHome, AppName)
	}

	return baseDir, nil
}

// EnsureConfigDir creates the config directory if it doesn't exist.
func EnsureConfigDir() (string, error) {
	configDir, err := ConfigDir()
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(configDir, 0700); err != nil {
		return "", fmt.Errorf("%w: create config directory: %w", ErrConfigDir, err)
	}

	return configDir, nil
}

// EnsureKeyringDir creates the keyring directory if it doesn't exist.
// This is used for the file-based keyring backend on headless systems.
func EnsureKeyringDir() (string, error) {
	configDir, err := ConfigDir()
	if err != nil {
		return "", err
	}

	keyringDir := filepath.Join(configDir, "keyring")

	if err := os.MkdirAll(keyringDir, 0700); err != nil {
		return "", fmt.Errorf("%w: create keyring directory: %w", ErrConfigDir, err)
	}

	return keyringDir, nil
}

// ConfigPath returns the path to the config file.
func ConfigPath() (string, error) {
	configDir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "config.json"), nil
}

// NormalizeEnvVarName converts a CLI name to an environment variable name.
// Example: "clickup-cli" → "CLICKUP_CLI"
func NormalizeEnvVarName(cliName string) string {
	return strings.ToUpper(strings.ReplaceAll(cliName, "-", "_"))
}
