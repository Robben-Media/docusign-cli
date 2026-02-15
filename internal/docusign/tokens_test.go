package docusign

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestTokenData_IsExpired(t *testing.T) {
	tests := []struct {
		name    string
		token   TokenData
		expired bool
	}{
		{
			name: "not expired",
			token: TokenData{
				ExpiresAt: time.Now().Unix() + 3600,
			},
			expired: false,
		},
		{
			name: "expired",
			token: TokenData{
				ExpiresAt: time.Now().Unix() - 100,
			},
			expired: true,
		},
		{
			name: "expires within 5 minutes",
			token: TokenData{
				ExpiresAt: time.Now().Unix() + 200,
			},
			expired: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.token.IsExpired() != tt.expired {
				t.Errorf("IsExpired() = %v, want %v", tt.token.IsExpired(), tt.expired)
			}
		})
	}
}

func TestWriteAndReadTokens(t *testing.T) {
	tmpDir := t.TempDir()
	tokenFile := filepath.Join(tmpDir, "tokens.json")

	// Override token file path for testing
	origHome := os.Getenv("HOME")

	t.Setenv("HOME", tmpDir)

	defer func() {
		t.Setenv("HOME", origHome)
	}()

	// Create the .docusign directory
	if err := os.MkdirAll(filepath.Join(tmpDir, ".docusign"), 0o700); err != nil {
		t.Fatalf("create dir: %v", err)
	}

	tokens := &TokenData{
		AccessToken:  "test-access-token",
		RefreshToken: "test-refresh-token",
		ExpiresIn:    28800,
		ExpiresAt:    time.Now().Unix() + 28800,
		TokenType:    "Bearer",
	}

	if err := WriteTokens(tokens); err != nil {
		t.Fatalf("WriteTokens: %v", err)
	}

	// Verify the file was written (the actual path may differ from tokenFile
	// since WriteTokens uses UserHomeDir internally)
	_ = tokenFile

	readTokens, err := ReadTokens()
	if err != nil {
		t.Fatalf("ReadTokens: %v", err)
	}

	if readTokens.AccessToken != tokens.AccessToken {
		t.Errorf("access token mismatch: got %q, want %q", readTokens.AccessToken, tokens.AccessToken)
	}

	if readTokens.RefreshToken != tokens.RefreshToken {
		t.Errorf("refresh token mismatch: got %q, want %q", readTokens.RefreshToken, tokens.RefreshToken)
	}

	if readTokens.TokenType != tokens.TokenType {
		t.Errorf("token type mismatch: got %q, want %q", readTokens.TokenType, tokens.TokenType)
	}
}

func TestReadTokens_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	_, err := ReadTokens()
	if err == nil {
		t.Fatal("expected error for missing token file")
	}
}

func TestRemoveTokens(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	docusignDir := filepath.Join(tmpDir, ".docusign")
	if err := os.MkdirAll(docusignDir, 0o700); err != nil {
		t.Fatalf("create dir: %v", err)
	}

	tokenFile := filepath.Join(docusignDir, "tokens.json")
	if err := os.WriteFile(tokenFile, []byte(`{}`), 0o600); err != nil {
		t.Fatalf("write token file: %v", err)
	}

	if err := RemoveTokens(); err != nil {
		t.Fatalf("RemoveTokens: %v", err)
	}

	if _, err := os.Stat(tokenFile); !os.IsNotExist(err) {
		t.Error("expected token file to be removed")
	}
}

func TestRemoveTokens_NotExist(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Should not error if file doesn't exist
	if err := RemoveTokens(); err != nil {
		t.Fatalf("RemoveTokens should not error for missing file: %v", err)
	}
}

func TestTokenFilePath(t *testing.T) {
	path, err := TokenFilePath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if path == "" {
		t.Fatal("expected non-empty path")
	}

	if filepath.Base(path) != "tokens.json" {
		t.Errorf("expected filename 'tokens.json', got %q", filepath.Base(path))
	}
}
