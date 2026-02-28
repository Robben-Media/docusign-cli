package docusign

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/builtbyrobben/docusign-cli/internal/api"
)

const (
	defaultOAuthBaseURL = "https://account.docusign.com"
	authPath            = "/oauth/auth"
	tokenPath           = "/oauth/token" //nolint:gosec // URL path, not a credential
	userinfoPath        = "/oauth/userinfo"
	defaultRedirectURI  = "http://localhost:3000/callback"
	oauthScopes         = "signature impersonation"
)

var (
	errEnvelopeIDRequired = errors.New("envelope ID is required")
	errDocumentIDRequired = errors.New("document ID is required")
	errTemplateIDRequired = errors.New("template ID is required")
	errVoidReasonReq      = errors.New("void reason is required")
	errTokenRefreshFailed = errors.New("token refresh failed")
	errNoAccountFound     = errors.New("no account found in userinfo response")
	errTokenExchange      = errors.New("token exchange failed")
	errUserinfoFailed     = errors.New("userinfo failed")
	errDocumentDownload   = errors.New("download document failed")
)

// Client wraps the API client with DocuSign-specific methods.
type Client struct {
	*api.Client
	accountID string
}

// NewClient creates a new DocuSign API client from stored tokens and credentials.
func NewClient(baseURI, accountID, accessToken string) *Client {
	apiBaseURL := baseURI + "/restapi/v2.1/accounts/" + accountID

	return &Client{
		Client: api.NewClient(accessToken,
			api.WithBaseURL(apiBaseURL),
			api.WithUserAgent("docusign-cli/1.0"),
		),
		accountID: accountID,
	}
}

// OAuthBaseURL returns the OAuth base URL from env or default.
func OAuthBaseURL() string {
	if v := os.Getenv("DOCUSIGN_OAUTH_BASE_URL"); v != "" {
		return v
	}

	return defaultOAuthBaseURL
}

// RedirectURI returns the redirect URI from env or default.
func RedirectURI() string {
	if v := os.Getenv("DOCUSIGN_REDIRECT_URI"); v != "" {
		return v
	}

	return defaultRedirectURI
}

// BuildAuthURL constructs the OAuth authorization URL.
func BuildAuthURL(integrationKey string) string {
	base := OAuthBaseURL() + authPath
	params := url.Values{
		"response_type": {"code"},
		"scope":         {oauthScopes},
		"client_id":     {integrationKey},
		"redirect_uri":  {RedirectURI()},
	}

	return base + "?" + params.Encode()
}

// TokenResponse represents the OAuth token endpoint response.
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

// ExchangeCode exchanges an authorization code for tokens.
func ExchangeCode(ctx context.Context, integrationKey, secretKey, code string) (*TokenResponse, error) {
	oauthBase := OAuthBaseURL()
	tokenURL := oauthBase + tokenPath

	data := url.Values{
		"grant_type":   {"authorization_code"},
		"code":         {code},
		"redirect_uri": {RedirectURI()},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("%w: create request: %w", errTokenExchange, err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Basic "+basicAuth(integrationKey, secretKey))

	client := &http.Client{Timeout: 30 * time.Second}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: execute request: %w", errTokenExchange, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)

		return nil, fmt.Errorf("%w: status %d: %s", errTokenExchange, resp.StatusCode, string(body))
	}

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("%w: decode response: %w", errTokenExchange, err)
	}

	return &tokenResp, nil
}

// RefreshAccessToken refreshes an expired access token using a refresh token.
func RefreshAccessToken(ctx context.Context, integrationKey, secretKey, refreshToken string) (*TokenResponse, error) {
	oauthBase := OAuthBaseURL()
	tokenURL := oauthBase + tokenPath

	data := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("%w: create request: %w", errTokenRefreshFailed, err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Basic "+basicAuth(integrationKey, secretKey))

	client := &http.Client{Timeout: 30 * time.Second}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: execute request: %w", errTokenRefreshFailed, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)

		return nil, fmt.Errorf("%w: status %d: %s", errTokenRefreshFailed, resp.StatusCode, string(body))
	}

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("%w: decode response: %w", errTokenRefreshFailed, err)
	}

	return &tokenResp, nil
}

// UserinfoResponse represents the /oauth/userinfo response.
type UserinfoResponse struct {
	Sub      string            `json:"sub"`
	Accounts []UserinfoAccount `json:"accounts"`
}

// UserinfoAccount represents an account in the userinfo response.
type UserinfoAccount struct {
	AccountID   string `json:"account_id"`
	IsDefault   bool   `json:"is_default"`
	AccountName string `json:"account_name"`
	BaseURI     string `json:"base_uri"`
}

// GetUserinfo calls the /oauth/userinfo endpoint.
func GetUserinfo(ctx context.Context, accessToken string) (*UserinfoResponse, error) {
	oauthBase := OAuthBaseURL()
	userinfoURL := oauthBase + userinfoPath

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, userinfoURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create userinfo request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{Timeout: 30 * time.Second}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute userinfo request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)

		return nil, fmt.Errorf("%w: status %d: %s", errUserinfoFailed, resp.StatusCode, string(body))
	}

	var info UserinfoResponse
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("decode userinfo response: %w", err)
	}

	return &info, nil
}

// FindDefaultAccount finds the default account from userinfo response.
func FindDefaultAccount(info *UserinfoResponse) (*UserinfoAccount, error) {
	if info == nil {
		return nil, errNoAccountFound
	}

	for i := range info.Accounts {
		if info.Accounts[i].IsDefault {
			return &info.Accounts[i], nil
		}
	}

	if len(info.Accounts) > 0 {
		return &info.Accounts[0], nil
	}

	return nil, errNoAccountFound
}

func basicAuth(username, password string) string {
	return base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
}

// Envelope types.

type Envelope struct {
	EnvelopeID     string `json:"envelope_id"`
	Status         string `json:"status"`
	EmailSubject   string `json:"email_subject"`
	SentDateTime   string `json:"sent_date_time,omitempty"`
	StatusDateTime string `json:"status_changed_date_time,omitempty"`
	CreatedAt      string `json:"created_date_time,omitempty"`
}

type EnvelopesListResponse struct {
	Envelopes     []Envelope `json:"envelopes"`
	ResultSetSize string     `json:"result_set_size"`
	TotalSetSize  string     `json:"total_set_size"`
	StartPosition string     `json:"start_position"`
	EndPosition   string     `json:"end_position"`
}

type EnvelopeDetail struct {
	EnvelopeID     string `json:"envelope_id"`
	Status         string `json:"status"`
	EmailSubject   string `json:"email_subject"`
	EmailBlurb     string `json:"email_blurb,omitempty"`
	SentDateTime   string `json:"sent_date_time,omitempty"`
	StatusDateTime string `json:"status_changed_date_time,omitempty"`
	CreatedAt      string `json:"created_date_time,omitempty"`
}

type CreateEnvelopeRequest struct {
	EmailSubject string      `json:"email_subject"`
	EmailBlurb   string      `json:"email_blurb,omitempty"`
	Status       string      `json:"status"`
	Documents    []Document  `json:"documents"`
	Recipients   *Recipients `json:"recipients"`
}

type Document struct {
	DocumentBase64 string `json:"document_base64"`
	Name           string `json:"name"`
	FileExtension  string `json:"file_extension"`
	DocumentID     string `json:"document_id"`
}

type Recipients struct {
	Signers []Signer `json:"signers"`
}

type Signer struct {
	Email        string `json:"email"`
	Name         string `json:"name"`
	RecipientID  string `json:"recipient_id"`
	RoutingOrder string `json:"routing_order,omitempty"`
	ClientUserID string `json:"client_user_id,omitempty"`
}

type AuditEvent struct {
	EventFields []EventField `json:"event_fields"`
}

type EventField struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type AuditEventsResponse struct {
	AuditEvents []AuditEvent `json:"audit_events"`
}

// Document types.

type DocumentInfo struct {
	DocumentID string `json:"document_id"`
	Name       string `json:"name"`
	Type       string `json:"type,omitempty"`
	URI        string `json:"uri,omitempty"`
	Order      string `json:"order,omitempty"`
	Pages      string `json:"pages,omitempty"`
}

type DocumentsListResponse struct {
	EnvelopeID        string         `json:"envelope_id"`
	EnvelopeDocuments []DocumentInfo `json:"envelope_documents"`
}

// Recipient types.

type RecipientInfo struct {
	RecipientID     string `json:"recipient_id"`
	RecipientIDGUID string `json:"recipient_id_guid,omitempty"`
	Name            string `json:"name"`
	Email           string `json:"email"`
	Status          string `json:"status,omitempty"`
	RoutingOrder    string `json:"routing_order,omitempty"`
	DeliveredAt     string `json:"delivered_date_time,omitempty"`
	SignedAt        string `json:"signed_date_time,omitempty"`
}

type RecipientsListResponse struct {
	Signers         []RecipientInfo `json:"signers"`
	CarbonCopies    []RecipientInfo `json:"carbon_copies,omitempty"`
	CertifiedDelivs []RecipientInfo `json:"certified_deliveries,omitempty"`
}

// Template types.

type Template struct {
	TemplateID   string `json:"template_id"`
	Name         string `json:"name"`
	Description  string `json:"description,omitempty"`
	Shared       string `json:"shared,omitempty"`
	Created      string `json:"created,omitempty"`
	LastModified string `json:"last_modified,omitempty"`
}

type TemplatesListResponse struct {
	EnvelopeTemplates []Template `json:"envelope_templates"`
	ResultSetSize     string     `json:"result_set_size"`
	TotalSetSize      string     `json:"total_set_size"`
}

// View types.

type RecipientViewRequest struct {
	ReturnURL            string `json:"return_url"`
	AuthenticationMethod string `json:"authentication_method"`
	Email                string `json:"email"`
	UserName             string `json:"user_name"`
	ClientUserID         string `json:"client_user_id,omitempty"`
}

type ViewURL struct {
	URL string `json:"url"`
}

// Envelopes service.

func (c *Client) ListEnvelopes(ctx context.Context, fromDate, status string, count int) (*EnvelopesListResponse, error) {
	params := url.Values{}

	if fromDate != "" {
		params.Set("from_date", fromDate)
	}

	if status != "" {
		params.Set("status", status)
	}

	if count > 0 {
		params.Set("count", fmt.Sprintf("%d", count))
	}

	path := "/envelopes"
	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	var result EnvelopesListResponse
	if err := c.Get(ctx, path, &result); err != nil {
		return nil, fmt.Errorf("list envelopes: %w", err)
	}

	return &result, nil
}

func (c *Client) GetEnvelope(ctx context.Context, envelopeID string) (*EnvelopeDetail, error) {
	if envelopeID == "" {
		return nil, errEnvelopeIDRequired
	}

	path := fmt.Sprintf("/envelopes/%s", url.PathEscape(envelopeID))

	var result EnvelopeDetail
	if err := c.Get(ctx, path, &result); err != nil {
		return nil, fmt.Errorf("get envelope: %w", err)
	}

	return &result, nil
}

func (c *Client) CreateEnvelope(ctx context.Context, req CreateEnvelopeRequest) (*EnvelopeDetail, error) {
	var result EnvelopeDetail
	if err := c.Post(ctx, "/envelopes", req, &result); err != nil {
		return nil, fmt.Errorf("create envelope: %w", err)
	}

	return &result, nil
}

func (c *Client) SendEnvelope(ctx context.Context, envelopeID string) error {
	if envelopeID == "" {
		return errEnvelopeIDRequired
	}

	path := fmt.Sprintf("/envelopes/%s", url.PathEscape(envelopeID))
	body := map[string]string{"status": "sent"}

	var result map[string]any
	if err := c.Put(ctx, path, body, &result); err != nil {
		return fmt.Errorf("send envelope: %w", err)
	}

	return nil
}

func (c *Client) VoidEnvelope(ctx context.Context, envelopeID, reason string) error {
	if envelopeID == "" {
		return errEnvelopeIDRequired
	}

	if reason == "" {
		return errVoidReasonReq
	}

	path := fmt.Sprintf("/envelopes/%s", url.PathEscape(envelopeID))
	body := map[string]string{
		"status":        "voided",
		"voided_reason": reason,
	}

	var result map[string]any
	if err := c.Put(ctx, path, body, &result); err != nil {
		return fmt.Errorf("void envelope: %w", err)
	}

	return nil
}

func (c *Client) GetAuditEvents(ctx context.Context, envelopeID string) (*AuditEventsResponse, error) {
	if envelopeID == "" {
		return nil, errEnvelopeIDRequired
	}

	path := fmt.Sprintf("/envelopes/%s/audit_events", url.PathEscape(envelopeID))

	var result AuditEventsResponse
	if err := c.Get(ctx, path, &result); err != nil {
		return nil, fmt.Errorf("get audit events: %w", err)
	}

	return &result, nil
}

// Documents service.

func (c *Client) ListDocuments(ctx context.Context, envelopeID string) (*DocumentsListResponse, error) {
	if envelopeID == "" {
		return nil, errEnvelopeIDRequired
	}

	path := fmt.Sprintf("/envelopes/%s/documents", url.PathEscape(envelopeID))

	var result DocumentsListResponse
	if err := c.Get(ctx, path, &result); err != nil {
		return nil, fmt.Errorf("list documents: %w", err)
	}

	return &result, nil
}

func (c *Client) DownloadDocument(ctx context.Context, envelopeID, documentID string) ([]byte, error) {
	if envelopeID == "" {
		return nil, errEnvelopeIDRequired
	}

	if documentID == "" {
		return nil, errDocumentIDRequired
	}

	path := fmt.Sprintf("/envelopes/%s/documents/%s", url.PathEscape(envelopeID), url.PathEscape(documentID))

	resp, err := c.Do(ctx, api.Request{Method: http.MethodGet, Path: path})
	if err != nil {
		return nil, fmt.Errorf("download document: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)

		return nil, fmt.Errorf("%w: status %d: %s", errDocumentDownload, resp.StatusCode, string(body))
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read document body: %w", err)
	}

	return data, nil
}

// Recipients service.

func (c *Client) ListRecipients(ctx context.Context, envelopeID string) (*RecipientsListResponse, error) {
	if envelopeID == "" {
		return nil, errEnvelopeIDRequired
	}

	path := fmt.Sprintf("/envelopes/%s/recipients", url.PathEscape(envelopeID))

	var result RecipientsListResponse
	if err := c.Get(ctx, path, &result); err != nil {
		return nil, fmt.Errorf("list recipients: %w", err)
	}

	return &result, nil
}

// Templates service.

func (c *Client) ListTemplates(ctx context.Context, search string, count int) (*TemplatesListResponse, error) {
	params := url.Values{}

	if search != "" {
		params.Set("search_text", search)
	}

	if count > 0 {
		params.Set("count", fmt.Sprintf("%d", count))
	}

	path := "/templates"
	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	var result TemplatesListResponse
	if err := c.Get(ctx, path, &result); err != nil {
		return nil, fmt.Errorf("list templates: %w", err)
	}

	return &result, nil
}

func (c *Client) GetTemplate(ctx context.Context, templateID string) (*Template, error) {
	if templateID == "" {
		return nil, errTemplateIDRequired
	}

	path := fmt.Sprintf("/templates/%s", url.PathEscape(templateID))

	var result Template
	if err := c.Get(ctx, path, &result); err != nil {
		return nil, fmt.Errorf("get template: %w", err)
	}

	return &result, nil
}

// Views service.

func (c *Client) CreateRecipientView(ctx context.Context, envelopeID string, req RecipientViewRequest) (*ViewURL, error) {
	if envelopeID == "" {
		return nil, errEnvelopeIDRequired
	}

	path := fmt.Sprintf("/envelopes/%s/views/recipient", url.PathEscape(envelopeID))

	var result ViewURL
	if err := c.Post(ctx, path, req, &result); err != nil {
		return nil, fmt.Errorf("create recipient view: %w", err)
	}

	return &result, nil
}
