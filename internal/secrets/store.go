package secrets

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/99designs/keyring"
	"golang.org/x/term"

	"github.com/builtbyrobben/cli-template/internal/config"
)

type Store interface {
	GetAPIKey() (string, error)
	SetAPIKey(key string) error
	DeleteAPIKey() error
	HasKey() (bool, error)
}

type KeyringStore struct {
	ring keyring.Keyring
}

const (
	apiKeyKey              = "api_key"
	keyringPasswordEnv     = "PLACEHOLDER_CLI_KEYRING_PASS"
	keyringBackendEnv      = "PLACEHOLDER_CLI_KEYRING_BACKEND"
	keyringOpenTimeout     = 5 * time.Second
)

var (
	errMissingAPIKey      = errors.New("missing API key")
	errNoTTY              = errors.New("no TTY available for keyring file backend password prompt")
	errInvalidKeyringBackend = errors.New("invalid keyring backend")
	errKeyringTimeout     = errors.New("keyring connection timed out")
)

type KeyringBackendInfo struct {
	Value  string
	Source string
}

const (
	keyringBackendSourceEnv     = "env"
	keyringBackendSourceConfig  = "config"
	keyringBackendSourceDefault = "default"
	keyringBackendAuto          = "auto"
)

func ResolveKeyringBackendInfo() (KeyringBackendInfo, error) {
	if v := normalizeKeyringBackend(os.Getenv(keyringBackendEnv)); v != "" {
		return KeyringBackendInfo{Value: v, Source: keyringBackendSourceEnv}, nil
	}

	// Could read from config file here if needed
	return KeyringBackendInfo{Value: keyringBackendAuto, Source: keyringBackendSourceDefault}, nil
}

func allowedBackends(info KeyringBackendInfo) ([]keyring.BackendType, error) {
	switch info.Value {
	case "", keyringBackendAuto:
		return nil, nil
	case "keychain":
		return []keyring.BackendType{keyring.KeychainBackend}, nil
	case "file":
		return []keyring.BackendType{keyring.FileBackend}, nil
	default:
		return nil, fmt.Errorf("%w: %q (expected %s, keychain, or file)", errInvalidKeyringBackend, info.Value, keyringBackendAuto)
	}
}

func fileKeyringPasswordFunc() keyring.PromptFunc {
	return fileKeyringPasswordFuncFrom(os.Getenv(keyringPasswordEnv), term.IsTerminal(int(os.Stdin.Fd())))
}

func fileKeyringPasswordFuncFrom(password string, isTTY bool) keyring.PromptFunc {
	if password != "" {
		return keyring.FixedStringPrompt(password)
	}

	if isTTY {
		return keyring.TerminalPrompt
	}

	return func(_ string) (string, error) {
		return "", fmt.Errorf("%w; set %s", errNoTTY, keyringPasswordEnv)
	}
}

func normalizeKeyringBackend(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func shouldForceFileBackend(goos string, backendInfo KeyringBackendInfo, dbusAddr string) bool {
	return goos == "linux" && backendInfo.Value == keyringBackendAuto && dbusAddr == ""
}

func shouldUseKeyringTimeout(goos string, backendInfo KeyringBackendInfo, dbusAddr string) bool {
	return goos == "linux" && backendInfo.Value == "auto" && dbusAddr != ""
}

func openKeyring() (keyring.Keyring, error) {
	keyringDir, err := config.EnsureKeyringDir()
	if err != nil {
		return nil, fmt.Errorf("ensure keyring dir: %w", err)
	}

	backendInfo, err := ResolveKeyringBackendInfo()
	if err != nil {
		return nil, err
	}

	backends, err := allowedBackends(backendInfo)
	if err != nil {
		return nil, err
	}

	dbusAddr := os.Getenv("DBUS_SESSION_BUS_ADDRESS")
	if shouldForceFileBackend(runtime.GOOS, backendInfo, dbusAddr) {
		backends = []keyring.BackendType{keyring.FileBackend}
	}

	cfg := keyring.Config{
		ServiceName:             config.AppName,
		KeychainTrustApplication: false,
		AllowedBackends:          backends,
		FileDir:                  keyringDir,
		FilePasswordFunc:         fileKeyringPasswordFunc(),
	}

	if shouldUseKeyringTimeout(runtime.GOOS, backendInfo, dbusAddr) {
		return openKeyringWithTimeout(cfg, keyringOpenTimeout)
	}

	ring, err := keyring.Open(cfg)
	if err != nil {
		return nil, fmt.Errorf("open keyring: %w", err)
	}

	return ring, nil
}

type keyringResult struct {
	ring keyring.Keyring
	err  error
}

func openKeyringWithTimeout(cfg keyring.Config, timeout time.Duration) (keyring.Keyring, error) {
	ch := make(chan keyringResult, 1)

	go func() {
		ring, err := keyring.Open(cfg)
		ch <- keyringResult{ring, err}
	}()

	select {
	case res := <-ch:
		if res.err != nil {
			return nil, fmt.Errorf("open keyring: %w", res.err)
		}

		return res.ring, nil
	case <-time.After(timeout):
		return nil, fmt.Errorf("%w after %v (D-Bus SecretService may be unresponsive); "+
			"set PLACEHOLDER_CLI_KEYRING_BACKEND=file and PLACEHOLDER_CLI_KEYRING_PASS=<password> to use encrypted file storage instead",
			errKeyringTimeout, timeout)
	}
}

func OpenDefault() (Store, error) {
	ring, err := openKeyring()
	if err != nil {
		return nil, err
	}

	return &KeyringStore{ring: ring}, nil
}

func (s *KeyringStore) GetAPIKey() (string, error) {
	item, err := s.ring.Get(apiKeyKey)
	if err != nil {
		return "", fmt.Errorf("read API key: %w", err)
	}

	return string(item.Data), nil
}

func (s *KeyringStore) SetAPIKey(key string) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return errMissingAPIKey
	}

	if err := s.ring.Set(keyring.Item{
		Key:  apiKeyKey,
		Data: []byte(key),
	}); err != nil {
		return fmt.Errorf("store API key: %w", err)
	}

	return nil
}

func (s *KeyringStore) DeleteAPIKey() error {
	if err := s.ring.Remove(apiKeyKey); err != nil && !errors.Is(err, keyring.ErrKeyNotFound) {
		return fmt.Errorf("delete API key: %w", err)
	}

	return nil
}

func (s *KeyringStore) HasKey() (bool, error) {
	_, err := s.ring.Get(apiKeyKey)
	if err != nil {
		if errors.Is(err, keyring.ErrKeyNotFound) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// GetSecret retrieves a generic secret by key.
func GetSecret(key string) ([]byte, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, errors.New("missing secret key")
	}

	ring, err := openKeyring()
	if err != nil {
		return nil, err
	}

	item, err := ring.Get(key)
	if err != nil {
		return nil, fmt.Errorf("read secret: %w", err)
	}

	return item.Data, nil
}

// SetSecret stores a generic secret by key.
func SetSecret(key string, value []byte) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return errors.New("missing secret key")
	}

	ring, err := openKeyring()
	if err != nil {
		return err
	}

	if err := ring.Set(keyring.Item{
		Key:  key,
		Data: value,
	}); err != nil {
		return fmt.Errorf("store secret: %w", err)
	}

	return nil
}
