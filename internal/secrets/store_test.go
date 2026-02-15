package secrets

import (
	"testing"
)

func TestNormalizeKeyringBackend(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"KEYCHAIN", "keychain"},
		{"Keychain", "keychain"},
		{"  keychain  ", "keychain"},
		{"FILE", "file"},
		{"File", "file"},
		{"  file  ", "file"},
		{"AUTO", "auto"},
		{"", ""},
		{"   ", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizeKeyringBackend(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeKeyringBackend(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestAllowedBackends(t *testing.T) {
	tests := []struct {
		name     string
		info     KeyringBackendInfo
		wantLen  int
		wantErr  bool
	}{
		{
			name:    "auto backend",
			info:    KeyringBackendInfo{Value: "auto", Source: "default"},
			wantLen: 0, // nil slice
			wantErr: false,
		},
		{
			name:    "empty backend",
			info:    KeyringBackendInfo{Value: "", Source: "default"},
			wantLen: 0, // nil slice
			wantErr: false,
		},
		{
			name:    "keychain backend",
			info:    KeyringBackendInfo{Value: "keychain", Source: "env"},
			wantLen: 1,
			wantErr: false,
		},
		{
			name:    "file backend",
			info:    KeyringBackendInfo{Value: "file", Source: "config"},
			wantLen: 1,
			wantErr: false,
		},
		{
			name:    "invalid backend",
			info:    KeyringBackendInfo{Value: "invalid", Source: "env"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backends, err := allowedBackends(tt.info)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if tt.wantLen == 0 && backends != nil {
				t.Errorf("expected nil slice, got %d backends", len(backends))
			} else if tt.wantLen > 0 && len(backends) != tt.wantLen {
				t.Errorf("expected %d backends, got %d", tt.wantLen, len(backends))
			}
		})
	}
}

func TestShouldForceFileBackend(t *testing.T) {
	tests := []struct {
		name       string
		goos       string
		backendInfo KeyringBackendInfo
		dbusAddr   string
		want       bool
	}{
		{
			name:       "Linux without D-Bus",
			goos:       "linux",
			backendInfo: KeyringBackendInfo{Value: "auto", Source: "default"},
			dbusAddr:   "",
			want:       true,
		},
		{
			name:       "Linux with D-Bus",
			goos:       "linux",
			backendInfo: KeyringBackendInfo{Value: "auto", Source: "default"},
			dbusAddr:   "unix:path=/run/user/1000/bus",
			want:       false,
		},
		{
			name:       "macOS without D-Bus",
			goos:       "darwin",
			backendInfo: KeyringBackendInfo{Value: "auto", Source: "default"},
			dbusAddr:   "",
			want:       false,
		},
		{
			name:       "Linux with explicit keychain backend",
			goos:       "linux",
			backendInfo: KeyringBackendInfo{Value: "keychain", Source: "env"},
			dbusAddr:   "",
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldForceFileBackend(tt.goos, tt.backendInfo, tt.dbusAddr)
			if result != tt.want {
				t.Errorf("shouldForceFileBackend() = %v, want %v", result, tt.want)
			}
		})
	}
}
