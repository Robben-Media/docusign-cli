package docusign

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

var (
	errTokenFileNotFound = errors.New("token file not found")
	errWriteTokenFile    = errors.New("write token file")
)

// TokenData represents the stored OAuth tokens.
type TokenData struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	ExpiresAt    int64  `json:"expires_at"`
	TokenType    string `json:"token_type"`
}

// IsExpired returns true if the access token has expired or will expire within 5 minutes.
func (t *TokenData) IsExpired() bool {
	return time.Now().Unix() >= t.ExpiresAt-300
}

// TokenFilePath returns the path to the token file (~/.docusign/tokens.json).
func TokenFilePath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home directory: %w", err)
	}

	return filepath.Join(homeDir, ".docusign", "tokens.json"), nil
}

// ReadTokens reads the token data from the token file.
func ReadTokens() (*TokenData, error) {
	path, err := TokenFilePath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path) //nolint:gosec // path is derived from os.UserHomeDir, not user input
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errTokenFileNotFound
		}

		return nil, fmt.Errorf("read token file: %w", err)
	}

	var tokens TokenData
	if err := json.Unmarshal(data, &tokens); err != nil {
		return nil, fmt.Errorf("parse token file: %w", err)
	}

	return &tokens, nil
}

// WriteTokens writes the token data to the token file.
func WriteTokens(tokens *TokenData) error {
	path, err := TokenFilePath()
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)

	if mkdirErr := os.MkdirAll(dir, 0o700); mkdirErr != nil {
		return fmt.Errorf("%w: create directory: %w", errWriteTokenFile, mkdirErr)
	}

	data, err := json.MarshalIndent(tokens, "", "  ")
	if err != nil {
		return fmt.Errorf("%w: marshal tokens: %w", errWriteTokenFile, err)
	}

	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("%w: %w", errWriteTokenFile, err)
	}

	return nil
}

// RemoveTokens deletes the token file.
func RemoveTokens() error {
	path, err := TokenFilePath()
	if err != nil {
		return err
	}

	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove token file: %w", err)
	}

	return nil
}
