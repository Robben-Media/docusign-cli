package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/builtbyrobben/cli-template/internal/outfmt"
	"github.com/builtbyrobben/cli-template/internal/secrets"
	"golang.org/x/term"
)

type AuthCmd struct {
	SetKey  AuthSetKeyCmd  `cmd:"" help:"Set API key (uses --stdin by default)"`
	Status  AuthStatusCmd  `cmd:"" help:"Show authentication status"`
	Remove  AuthRemoveCmd  `cmd:"" help:"Remove stored credentials"`
}

type AuthSetKeyCmd struct {
	Stdin bool `help:"Read API key from stdin (default: true)" default:"true"`
	Key    string `arg:"" optional:"" help:"API key (discouraged; exposes in shell history)"`
}

func (cmd *AuthSetKeyCmd) Run(ctx context.Context) error {
	var apiKey string

	// Priority: argument > stdin
	if cmd.Key != "" {
		// Warn about shell history exposure
		fmt.Fprintln(os.Stderr, "Warning: passing keys as arguments exposes them in shell history. Use --stdin instead.")
		apiKey = strings.TrimSpace(cmd.Key)
	} else if term.IsTerminal(int(os.Stdin.Fd())) {
		// Interactive prompt
		fmt.Fprint(os.Stderr, "Enter API key: ")
		byteKey, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Fprintln(os.Stderr) // New line after password input
		if err != nil {
			return fmt.Errorf("read API key: %w", err)
		}
		apiKey = strings.TrimSpace(string(byteKey))
	} else {
		// Read from stdin (piped)
		byteKey, err := io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("read API key from stdin: %w", err)
		}
		apiKey = strings.TrimSpace(string(byteKey))
	}

	if apiKey == "" {
		return fmt.Errorf("API key cannot be empty")
	}

	store, err := secrets.OpenDefault()
	if err != nil {
		return fmt.Errorf("open credential store: %w", err)
	}

	if err := store.SetAPIKey(apiKey); err != nil {
		return fmt.Errorf("store API key: %w", err)
	}

	if outfmt.IsJSON(ctx) {
		outfmt.WriteJSON(os.Stdout, map[string]string{
			"status": "success",
			"message": "API key stored in keyring",
		})
	} else {
		fmt.Fprintln(os.Stderr, "API key stored in keyring")
	}

	return nil
}

type AuthStatusCmd struct{}

func (cmd *AuthStatusCmd) Run(ctx context.Context) error {
	store, err := secrets.OpenDefault()
	if err != nil {
		return fmt.Errorf("open credential store: %w", err)
	}

	hasKey, err := store.HasKey()
	if err != nil {
		return fmt.Errorf("check API key: %w", err)
	}

	// Check environment variable override
	envKey := os.Getenv("PLACEHOLDER_CLI_API_KEY")
	envOverride := envKey != ""

	status := map[string]any{
		"has_key":        hasKey,
		"env_override":   envOverride,
		"storage_backend": "keyring",
	}

	if hasKey && !envOverride {
		// Show redacted key
		key, err := store.GetAPIKey()
		if err == nil && len(key) > 8 {
			status["key_redacted"] = key[:4] + "..." + key[len(key)-4:]
		}
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(os.Stdout, status)
	}

	// Human-readable output
	fmt.Fprintf(os.Stderr, "Storage: %s\n", status["storage_backend"])
	if envOverride {
		fmt.Fprintln(os.Stderr, "Status: Using PLACEHOLDER_CLI_API_KEY environment variable")
	} else if hasKey {
		fmt.Fprintln(os.Stderr, "Status: Authenticated")
		if redacted, ok := status["key_redacted"].(string); ok {
			fmt.Fprintf(os.Stderr, "Key: %s\n", redacted)
		}
	} else {
		fmt.Fprintln(os.Stderr, "Status: Not authenticated")
		fmt.Fprintln(os.Stderr, "Run: placeholder-cli auth set-key --stdin")
	}

	return nil
}

type AuthRemoveCmd struct{}

func (cmd *AuthRemoveCmd) Run(ctx context.Context) error {
	store, err := secrets.OpenDefault()
	if err != nil {
		return fmt.Errorf("open credential store: %w", err)
	}

	if err := store.DeleteAPIKey(); err != nil {
		return fmt.Errorf("remove API key: %w", err)
	}

	if outfmt.IsJSON(ctx) {
		outfmt.WriteJSON(os.Stdout, map[string]string{
			"status": "success",
			"message": "API key removed",
		})
	} else {
		fmt.Fprintln(os.Stderr, "API key removed")
	}

	return nil
}
