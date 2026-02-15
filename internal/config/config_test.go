package config

import (
	"runtime"
	"strings"
	"testing"
)

func TestAppName(t *testing.T) {
	if AppName != "docusign-cli" {
		t.Errorf("AppName = %q, want 'docusign-cli'", AppName)
	}
}

func TestConfigDir(t *testing.T) {
	dir, err := ConfigDir()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if dir == "" {
		t.Fatal("ConfigDir returned empty string")
	}

	if !strings.Contains(dir, AppName) {
		t.Errorf("expected config dir to contain %q, got %q", AppName, dir)
	}

	switch runtime.GOOS {
	case "darwin":
		if !strings.Contains(dir, "Library/Application Support") {
			t.Errorf("macOS config dir should contain 'Library/Application Support', got %q", dir)
		}
	case "linux":
		if !strings.Contains(dir, ".config") && !strings.Contains(dir, "XDG") {
			t.Logf("Linux config dir: %s (custom XDG_CONFIG_HOME may be set)", dir)
		}
	}
}

func TestEnsureConfigDir(t *testing.T) {
	dir, err := EnsureConfigDir()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if dir == "" {
		t.Fatal("EnsureConfigDir returned empty string")
	}
}

func TestEnsureKeyringDir(t *testing.T) {
	dir, err := EnsureKeyringDir()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.HasSuffix(dir, "keyring") {
		t.Errorf("expected keyring dir to end with 'keyring', got %q", dir)
	}
}

func TestConfigPath(t *testing.T) {
	path, err := ConfigPath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.HasSuffix(path, "config.json") {
		t.Errorf("expected config path to end with 'config.json', got %q", path)
	}
}

func TestNormalizeEnvVarName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"docusign-cli", "DOCUSIGN_CLI"},
		{"surfer-cli", "SURFER_CLI"},
		{"exa-cli", "EXA_CLI"},
		{"simple", "SIMPLE"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := NormalizeEnvVarName(tt.input)
			if got != tt.want {
				t.Errorf("NormalizeEnvVarName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
