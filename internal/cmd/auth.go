package cmd

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"golang.org/x/term"

	"github.com/builtbyrobben/docusign-cli/internal/docusign"
	"github.com/builtbyrobben/docusign-cli/internal/outfmt"
	"github.com/builtbyrobben/docusign-cli/internal/secrets"
)

const (
	integrationKeyName = "integration-key"
	secretKeyName      = "secret-key"
	accountIDKeyName   = "account-id"
	baseURIKeyName     = "base-uri"
)

type AuthCmd struct {
	Login          AuthLoginCmd          `cmd:"" help:"Login via OAuth 2.0 authorization code flow"`
	SetCredentials AuthSetCredentialsCmd `cmd:"" help:"Set integration key and secret key"`
	Status         AuthStatusCmd         `cmd:"" help:"Show authentication status"`
	Remove         AuthRemoveCmd         `cmd:"" help:"Remove all stored credentials and tokens"`
}

type AuthLoginCmd struct{}

func (cmd *AuthLoginCmd) Run(ctx context.Context) error {
	integrationKey, secretKey, err := getOAuthCredentials()
	if err != nil {
		return err
	}

	authURL := docusign.BuildAuthURL(integrationKey)
	fmt.Fprintln(os.Stderr, "Open this URL in your browser to authorize:")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, authURL)
	fmt.Fprintln(os.Stderr)
	fmt.Fprint(os.Stderr, "Paste the authorization code: ")

	reader := bufio.NewReader(os.Stdin)

	code, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("read authorization code: %w", err)
	}

	code = strings.TrimSpace(code)
	if code == "" {
		return fmt.Errorf("authorization code cannot be empty")
	}

	fmt.Fprintln(os.Stderr, "Exchanging code for tokens...")

	tokenResp, err := docusign.ExchangeCode(ctx, integrationKey, secretKey, code)
	if err != nil {
		return err
	}

	tokens := &docusign.TokenData{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		ExpiresIn:    tokenResp.ExpiresIn,
		ExpiresAt:    time.Now().Unix() + tokenResp.ExpiresIn,
		TokenType:    tokenResp.TokenType,
	}

	writeErr := docusign.WriteTokens(tokens)
	if writeErr != nil {
		return writeErr
	}

	fmt.Fprintln(os.Stderr, "Discovering account info...")

	info, err := docusign.GetUserinfo(ctx, tokens.AccessToken)
	if err != nil {
		return err
	}

	account, err := docusign.FindDefaultAccount(info)
	if err != nil {
		return err
	}

	if storeErr := secrets.SetSecret(accountIDKeyName, []byte(account.AccountID)); storeErr != nil {
		return fmt.Errorf("store account ID: %w", storeErr)
	}

	if storeErr := secrets.SetSecret(baseURIKeyName, []byte(account.BaseURI)); storeErr != nil {
		return fmt.Errorf("store base URI: %w", storeErr)
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(os.Stdout, map[string]string{
			"status":     "success",
			"account_id": account.AccountID,
			"base_uri":   account.BaseURI,
		})
	}
	if outfmt.IsPlain(ctx) {
		return outfmt.WritePlain(os.Stdout,
			[]string{"STATUS", "ACCOUNT_ID", "BASE_URI"},
			[][]string{{"success", account.AccountID, account.BaseURI}},
		)
	}

	fmt.Fprintln(os.Stderr, "Login successful!")
	fmt.Fprintf(os.Stderr, "Account ID: %s\n", account.AccountID)
	fmt.Fprintf(os.Stderr, "Account:    %s\n", account.AccountName)
	fmt.Fprintf(os.Stderr, "Base URI:   %s\n", account.BaseURI)

	return nil
}

type AuthSetCredentialsCmd struct{}

func (cmd *AuthSetCredentialsCmd) Run(ctx context.Context) error {
	var integrationKey, secretKey string

	if envIK := os.Getenv("DOCUSIGN_INTEGRATION_KEY"); envIK != "" {
		integrationKey = envIK
	}

	if envSK := os.Getenv("DOCUSIGN_SECRET_KEY"); envSK != "" {
		secretKey = envSK
	}

	if integrationKey == "" {
		if term.IsTerminal(int(os.Stdin.Fd())) {
			fmt.Fprint(os.Stderr, "Enter integration key (client ID): ")

			byteKey, readErr := term.ReadPassword(int(os.Stdin.Fd()))
			fmt.Fprintln(os.Stderr)

			if readErr != nil {
				return fmt.Errorf("read integration key: %w", readErr)
			}

			integrationKey = strings.TrimSpace(string(byteKey))
		} else {
			byteKey, readErr := io.ReadAll(os.Stdin)
			if readErr != nil {
				return fmt.Errorf("read integration key from stdin: %w", readErr)
			}

			parts := strings.SplitN(strings.TrimSpace(string(byteKey)), "\n", 2)
			integrationKey = strings.TrimSpace(parts[0])

			if len(parts) > 1 {
				secretKey = strings.TrimSpace(parts[1])
			}
		}
	}

	if secretKey == "" && term.IsTerminal(int(os.Stdin.Fd())) {
		fmt.Fprint(os.Stderr, "Enter secret key: ")

		byteKey, readErr := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Fprintln(os.Stderr)

		if readErr != nil {
			return fmt.Errorf("read secret key: %w", readErr)
		}

		secretKey = strings.TrimSpace(string(byteKey))
	}

	if integrationKey == "" {
		return fmt.Errorf("integration key cannot be empty")
	}

	if secretKey == "" {
		if !term.IsTerminal(int(os.Stdin.Fd())) {
			return fmt.Errorf("secret key cannot be empty (pipe both values as two lines: integration-key\\nsecret-key)")
		}

		return fmt.Errorf("secret key cannot be empty")
	}

	if err := secrets.SetSecret(integrationKeyName, []byte(integrationKey)); err != nil {
		return fmt.Errorf("store integration key: %w", err)
	}

	if err := secrets.SetSecret(secretKeyName, []byte(secretKey)); err != nil {
		return fmt.Errorf("store secret key: %w", err)
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(os.Stdout, map[string]string{
			"status":  "success",
			"message": "Credentials stored in keyring",
		})
	}
	if outfmt.IsPlain(ctx) {
		return outfmt.WritePlain(os.Stdout,
			[]string{"STATUS", "MESSAGE"},
			[][]string{{"success", "Credentials stored in keyring"}},
		)
	}

	fmt.Fprintln(os.Stderr, "Credentials stored in keyring")

	return nil
}

type AuthStatusCmd struct{}

func (cmd *AuthStatusCmd) Run(ctx context.Context) error {
	hasIK, _ := secrets.HasSecret(integrationKeyName)
	hasSK, _ := secrets.HasSecret(secretKeyName)
	hasAcct, _ := secrets.HasSecret(accountIDKeyName)

	envIK := os.Getenv("DOCUSIGN_INTEGRATION_KEY") != ""
	envSK := os.Getenv("DOCUSIGN_SECRET_KEY") != ""
	envAcct := os.Getenv("DOCUSIGN_ACCOUNT_ID") != ""

	tokens, tokenErr := docusign.ReadTokens()
	hasTokens := tokenErr == nil

	status := map[string]any{
		"has_integration_key": hasIK || envIK,
		"has_secret_key":      hasSK || envSK,
		"has_account_id":      hasAcct || envAcct,
		"has_tokens":          hasTokens,
		"env_integration_key": envIK,
		"env_secret_key":      envSK,
		"env_account_id":      envAcct,
	}

	if envAcct {
		status["account_id"] = os.Getenv("DOCUSIGN_ACCOUNT_ID")
	} else if hasAcct {
		acctID, acctErr := secrets.GetSecret(accountIDKeyName)
		if acctErr == nil {
			status["account_id"] = string(acctID)
		}
	}

	if hasTokens {
		status["token_expired"] = tokens.IsExpired()

		if len(tokens.AccessToken) > 8 {
			status["access_token_redacted"] = tokens.AccessToken[:4] + "..." + tokens.AccessToken[len(tokens.AccessToken)-4:]
		}
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(os.Stdout, status)
	}
	if outfmt.IsPlain(ctx) {
		acctID := ""
		if v, ok := status["account_id"].(string); ok {
			acctID = v
		}
		tokenStatus := "none"
		if hasTokens {
			if tokens.IsExpired() {
				tokenStatus = "expired"
			} else {
				tokenStatus = "valid"
			}
		}
		return outfmt.WritePlain(os.Stdout,
			[]string{"HAS_INTEGRATION_KEY", "HAS_SECRET_KEY", "ACCOUNT_ID", "TOKENS"},
			[][]string{{fmt.Sprintf("%v", hasIK || envIK), fmt.Sprintf("%v", hasSK || envSK), acctID, tokenStatus}},
		)
	}

	fmt.Fprintln(os.Stdout, "DocuSign CLI Auth Status")
	fmt.Fprintln(os.Stdout, "------------------------")

	printIntegrationKeyStatus(envIK, hasIK)
	printSecretKeyStatus(envSK, hasSK)

	switch {
	case envAcct:
		fmt.Fprintf(os.Stdout, "Account ID:      %s (env)\n", os.Getenv("DOCUSIGN_ACCOUNT_ID"))
	case hasAcct:
		if acctID, ok := status["account_id"].(string); ok {
			fmt.Fprintf(os.Stdout, "Account ID:      %s\n", acctID)
		}
	default:
		fmt.Fprintln(os.Stdout, "Account ID:      Not set")
	}

	printTokenStatus(hasTokens, tokens, status)

	return nil
}

func printIntegrationKeyStatus(envIK, hasIK bool) {
	switch {
	case envIK:
		fmt.Fprintln(os.Stdout, "Integration Key: Set via DOCUSIGN_INTEGRATION_KEY")
	case hasIK:
		fmt.Fprintln(os.Stdout, "Integration Key: Stored in keyring")
	default:
		fmt.Fprintln(os.Stdout, "Integration Key: Not set")
	}
}

func printSecretKeyStatus(envSK, hasSK bool) {
	switch {
	case envSK:
		fmt.Fprintln(os.Stdout, "Secret Key:      Set via DOCUSIGN_SECRET_KEY")
	case hasSK:
		fmt.Fprintln(os.Stdout, "Secret Key:      Stored in keyring")
	default:
		fmt.Fprintln(os.Stdout, "Secret Key:      Not set")
	}
}

func printTokenStatus(hasTokens bool, tokens *docusign.TokenData, status map[string]any) {
	if hasTokens {
		if tokens.IsExpired() {
			fmt.Fprintln(os.Stdout, "Tokens:          Expired (will auto-refresh)")
		} else {
			fmt.Fprintln(os.Stdout, "Tokens:          Valid")
		}

		if redacted, ok := status["access_token_redacted"].(string); ok {
			fmt.Fprintf(os.Stdout, "Access Token:    %s\n", redacted)
		}
	} else {
		fmt.Fprintln(os.Stdout, "Tokens:          Not found (run: docusign-cli auth login)")
	}
}

type AuthRemoveCmd struct{}

func (cmd *AuthRemoveCmd) Run(ctx context.Context) error {
	if err := secrets.DeleteSecret(integrationKeyName); err != nil {
		return fmt.Errorf("remove integration key: %w", err)
	}

	if err := secrets.DeleteSecret(secretKeyName); err != nil {
		return fmt.Errorf("remove secret key: %w", err)
	}

	if err := secrets.DeleteSecret(accountIDKeyName); err != nil {
		return fmt.Errorf("remove account ID: %w", err)
	}

	if err := secrets.DeleteSecret(baseURIKeyName); err != nil {
		return fmt.Errorf("remove base URI: %w", err)
	}

	if err := docusign.RemoveTokens(); err != nil {
		return fmt.Errorf("remove tokens: %w", err)
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(os.Stdout, map[string]string{
			"status":  "success",
			"message": "All credentials and tokens removed",
		})
	}
	if outfmt.IsPlain(ctx) {
		return outfmt.WritePlain(os.Stdout,
			[]string{"STATUS", "MESSAGE"},
			[][]string{{"success", "All credentials and tokens removed"}},
		)
	}

	fmt.Fprintln(os.Stderr, "All credentials and tokens removed")

	return nil
}

// getOAuthCredentials returns integration key and secret key from env or keyring.
func getOAuthCredentials() (string, string, error) {
	integrationKey := os.Getenv("DOCUSIGN_INTEGRATION_KEY")
	if integrationKey == "" {
		ik, err := secrets.GetSecret(integrationKeyName)
		if err != nil {
			return "", "", fmt.Errorf("get integration key: %w (run 'docusign-cli auth set-credentials' first)", err)
		}

		integrationKey = string(ik)
	}

	secretKey := os.Getenv("DOCUSIGN_SECRET_KEY")
	if secretKey == "" {
		sk, err := secrets.GetSecret(secretKeyName)
		if err != nil {
			return "", "", fmt.Errorf("get secret key: %w (run 'docusign-cli auth set-credentials' first)", err)
		}

		secretKey = string(sk)
	}

	return integrationKey, secretKey, nil
}

// getDocuSignClient creates an authenticated DocuSign client, refreshing tokens if needed.
func getDocuSignClient(ctx context.Context) (*docusign.Client, error) {
	tokens, err := docusign.ReadTokens()
	if err != nil {
		return nil, fmt.Errorf("read tokens: %w (run 'docusign-cli auth login' first)", err)
	}

	if tokens.IsExpired() {
		integrationKey, secretKey, credErr := getOAuthCredentials()
		if credErr != nil {
			return nil, credErr
		}

		tokenResp, refreshErr := docusign.RefreshAccessToken(ctx, integrationKey, secretKey, tokens.RefreshToken)
		if refreshErr != nil {
			return nil, fmt.Errorf("refresh token: %w (run 'docusign-cli auth login' to re-authenticate)", refreshErr)
		}

		tokens.AccessToken = tokenResp.AccessToken
		tokens.ExpiresIn = tokenResp.ExpiresIn
		tokens.ExpiresAt = time.Now().Unix() + tokenResp.ExpiresIn

		if tokenResp.RefreshToken != "" {
			tokens.RefreshToken = tokenResp.RefreshToken
		}

		if writeErr := docusign.WriteTokens(tokens); writeErr != nil {
			return nil, writeErr
		}

		fmt.Fprintln(os.Stderr, "Token refreshed")
	}

	accountID := os.Getenv("DOCUSIGN_ACCOUNT_ID")
	if accountID == "" {
		accountIDBytes, err := secrets.GetSecret(accountIDKeyName)
		if err != nil {
			return nil, fmt.Errorf("get account ID: %w (run 'docusign-cli auth login' first)", err)
		}
		accountID = string(accountIDBytes)
	}

	baseURI := os.Getenv("DOCUSIGN_BASE_URI")
	if baseURI == "" {
		baseURIBytes, err := secrets.GetSecret(baseURIKeyName)
		if err != nil {
			return nil, fmt.Errorf("get base URI: %w (run 'docusign-cli auth login' first)", err)
		}
		baseURI = string(baseURIBytes)
	}

	return docusign.NewClient(baseURI, accountID, tokens.AccessToken), nil
}
