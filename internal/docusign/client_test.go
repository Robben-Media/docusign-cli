package docusign

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/builtbyrobben/docusign-cli/internal/api"
)

func newTestClient(server *httptest.Server) *Client {
	return &Client{
		Client: api.NewClient("test-token",
			api.WithBaseURL(server.URL),
			api.WithUserAgent("docusign-cli/test"),
		),
		accountID: "test-account",
	}
}

func TestListEnvelopes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/envelopes" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		if r.URL.Query().Get("count") != "5" {
			t.Errorf("expected count=5, got %s", r.URL.Query().Get("count"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(EnvelopesListResponse{
			Envelopes: []Envelope{
				{EnvelopeID: "env-1", Status: "sent", EmailSubject: "Test"},
				{EnvelopeID: "env-2", Status: "completed", EmailSubject: "Test 2"},
			},
			ResultSetSize: "2",
			TotalSetSize:  "2",
		})
	}))
	defer server.Close()

	client := newTestClient(server)

	result, err := client.ListEnvelopes(context.Background(), "", "", 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Envelopes) != 2 {
		t.Fatalf("expected 2 envelopes, got %d", len(result.Envelopes))
	}

	if result.Envelopes[0].EnvelopeID != "env-1" {
		t.Errorf("expected first envelope ID 'env-1', got %q", result.Envelopes[0].EnvelopeID)
	}
}

func TestGetEnvelope(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/envelopes/env-123" {
			t.Errorf("expected path /envelopes/env-123, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(EnvelopeDetail{
			EnvelopeID:   "env-123",
			Status:       "sent",
			EmailSubject: "Test Envelope",
		})
	}))
	defer server.Close()

	client := newTestClient(server)

	result, err := client.GetEnvelope(context.Background(), "env-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.EnvelopeID != "env-123" {
		t.Errorf("expected ID 'env-123', got %q", result.EnvelopeID)
	}

	if result.Status != "sent" {
		t.Errorf("expected status 'sent', got %q", result.Status)
	}
}

func TestGetEnvelope_EmptyID(t *testing.T) {
	client := &Client{Client: api.NewClient("key"), accountID: "acct"}

	_, err := client.GetEnvelope(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty ID")
	}
}

func TestCreateEnvelope(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		if r.URL.Path != "/envelopes" {
			t.Errorf("expected path /envelopes, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(EnvelopeDetail{
			EnvelopeID: "new-env",
			Status:     "sent",
		})
	}))
	defer server.Close()

	client := newTestClient(server)

	result, err := client.CreateEnvelope(context.Background(), CreateEnvelopeRequest{
		EmailSubject: "Test",
		Status:       "sent",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.EnvelopeID != "new-env" {
		t.Errorf("expected ID 'new-env', got %q", result.EnvelopeID)
	}
}

func TestSendEnvelope(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "sent"})
	}))
	defer server.Close()

	client := newTestClient(server)

	err := client.SendEnvelope(context.Background(), "env-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSendEnvelope_EmptyID(t *testing.T) {
	client := &Client{Client: api.NewClient("key"), accountID: "acct"}

	err := client.SendEnvelope(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty ID")
	}
}

func TestVoidEnvelope(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "voided"})
	}))
	defer server.Close()

	client := newTestClient(server)

	err := client.VoidEnvelope(context.Background(), "env-123", "test reason")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVoidEnvelope_EmptyReason(t *testing.T) {
	client := &Client{Client: api.NewClient("key"), accountID: "acct"}

	err := client.VoidEnvelope(context.Background(), "env-123", "")
	if err == nil {
		t.Fatal("expected error for empty reason")
	}
}

func TestGetAuditEvents(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/envelopes/env-123/audit_events" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(AuditEventsResponse{
			AuditEvents: []AuditEvent{
				{EventFields: []EventField{{Name: "Action", Value: "Sent"}}},
			},
		})
	}))
	defer server.Close()

	client := newTestClient(server)

	result, err := client.GetAuditEvents(context.Background(), "env-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.AuditEvents) != 1 {
		t.Fatalf("expected 1 audit event, got %d", len(result.AuditEvents))
	}
}

func TestListDocuments(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/envelopes/env-123/documents" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(DocumentsListResponse{
			EnvelopeID: "env-123",
			EnvelopeDocuments: []DocumentInfo{
				{DocumentID: "1", Name: "contract.pdf"},
			},
		})
	}))
	defer server.Close()

	client := newTestClient(server)

	result, err := client.ListDocuments(context.Background(), "env-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.EnvelopeDocuments) != 1 {
		t.Fatalf("expected 1 document, got %d", len(result.EnvelopeDocuments))
	}

	if result.EnvelopeDocuments[0].Name != "contract.pdf" {
		t.Errorf("expected name 'contract.pdf', got %q", result.EnvelopeDocuments[0].Name)
	}
}

func TestListDocuments_EmptyID(t *testing.T) {
	client := &Client{Client: api.NewClient("key"), accountID: "acct"}

	_, err := client.ListDocuments(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty envelope ID")
	}
}

func TestDownloadDocument(t *testing.T) {
	pdfContent := []byte("%PDF-1.4 test content")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/envelopes/env-123/documents/1" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/pdf")
		w.Write(pdfContent)
	}))
	defer server.Close()

	client := newTestClient(server)

	data, err := client.DownloadDocument(context.Background(), "env-123", "1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if string(data) != string(pdfContent) {
		t.Errorf("expected PDF content, got %q", string(data))
	}
}

func TestDownloadDocument_EmptyIDs(t *testing.T) {
	client := &Client{Client: api.NewClient("key"), accountID: "acct"}

	_, err := client.DownloadDocument(context.Background(), "", "1")
	if err == nil {
		t.Fatal("expected error for empty envelope ID")
	}

	_, err = client.DownloadDocument(context.Background(), "env-123", "")
	if err == nil {
		t.Fatal("expected error for empty document ID")
	}
}

func TestListRecipients(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/envelopes/env-123/recipients" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(RecipientsListResponse{
			Signers: []RecipientInfo{
				{RecipientID: "1", Name: "John Doe", Email: "john@example.com", Status: "sent"},
			},
		})
	}))
	defer server.Close()

	client := newTestClient(server)

	result, err := client.ListRecipients(context.Background(), "env-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Signers) != 1 {
		t.Fatalf("expected 1 signer, got %d", len(result.Signers))
	}

	if result.Signers[0].Email != "john@example.com" {
		t.Errorf("expected email 'john@example.com', got %q", result.Signers[0].Email)
	}
}

func TestListTemplates(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/templates" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		if r.URL.Query().Get("search_text") != "NDA" {
			t.Errorf("expected search_text=NDA, got %s", r.URL.Query().Get("search_text"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(TemplatesListResponse{
			EnvelopeTemplates: []Template{
				{TemplateID: "tmpl-1", Name: "NDA Template"},
			},
			ResultSetSize: "1",
			TotalSetSize:  "1",
		})
	}))
	defer server.Close()

	client := newTestClient(server)

	result, err := client.ListTemplates(context.Background(), "NDA", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.EnvelopeTemplates) != 1 {
		t.Fatalf("expected 1 template, got %d", len(result.EnvelopeTemplates))
	}

	if result.EnvelopeTemplates[0].Name != "NDA Template" {
		t.Errorf("expected name 'NDA Template', got %q", result.EnvelopeTemplates[0].Name)
	}
}

func TestGetTemplate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/templates/tmpl-123" {
			t.Errorf("expected path /templates/tmpl-123, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Template{
			TemplateID: "tmpl-123",
			Name:       "Test Template",
		})
	}))
	defer server.Close()

	client := newTestClient(server)

	result, err := client.GetTemplate(context.Background(), "tmpl-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.TemplateID != "tmpl-123" {
		t.Errorf("expected ID 'tmpl-123', got %q", result.TemplateID)
	}
}

func TestGetTemplate_EmptyID(t *testing.T) {
	client := &Client{Client: api.NewClient("key"), accountID: "acct"}

	_, err := client.GetTemplate(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty ID")
	}
}

func TestCreateRecipientView(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		if r.URL.Path != "/envelopes/env-123/views/recipient" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ViewURL{
			URL: "https://demo.docusign.net/signing/...",
		})
	}))
	defer server.Close()

	client := newTestClient(server)

	result, err := client.CreateRecipientView(context.Background(), "env-123", RecipientViewRequest{
		ReturnURL:            "https://example.com/return",
		AuthenticationMethod: "none",
		Email:                "john@example.com",
		UserName:             "John Doe",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.URL == "" {
		t.Error("expected non-empty URL")
	}
}

func TestCreateRecipientView_EmptyEnvelopeID(t *testing.T) {
	client := &Client{Client: api.NewClient("key"), accountID: "acct"}

	_, err := client.CreateRecipientView(context.Background(), "", RecipientViewRequest{})
	if err == nil {
		t.Fatal("expected error for empty envelope ID")
	}
}

func TestBuildAuthURL(t *testing.T) {
	t.Setenv("DOCUSIGN_OAUTH_BASE_URL", "https://account-d.docusign.com")

	authURL := BuildAuthURL("test-integration-key")

	if authURL == "" {
		t.Fatal("expected non-empty auth URL")
	}

	if !strings.Contains(authURL, "test-integration-key") {
		t.Errorf("expected URL to contain integration key, got %s", authURL)
	}

	if !strings.Contains(authURL, "response_type=code") {
		t.Errorf("expected URL to contain response_type=code, got %s", authURL)
	}
}

func TestBasicAuth(t *testing.T) {
	result := basicAuth("user", "pass")
	if result == "" {
		t.Fatal("expected non-empty basic auth string")
	}
}

func TestFindDefaultAccount(t *testing.T) {
	info := &UserinfoResponse{
		Accounts: []UserinfoAccount{
			{AccountID: "acct-1", IsDefault: false, AccountName: "Secondary"},
			{AccountID: "acct-2", IsDefault: true, AccountName: "Primary"},
		},
	}

	account, err := FindDefaultAccount(info)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if account.AccountID != "acct-2" {
		t.Errorf("expected default account 'acct-2', got %q", account.AccountID)
	}
}

func TestFindDefaultAccount_NoDefault(t *testing.T) {
	info := &UserinfoResponse{
		Accounts: []UserinfoAccount{
			{AccountID: "acct-1", IsDefault: false},
		},
	}

	account, err := FindDefaultAccount(info)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if account.AccountID != "acct-1" {
		t.Errorf("expected first account 'acct-1', got %q", account.AccountID)
	}
}

func TestFindDefaultAccount_NoAccounts(t *testing.T) {
	info := &UserinfoResponse{Accounts: []UserinfoAccount{}}

	_, err := FindDefaultAccount(info)
	if err == nil {
		t.Fatal("expected error for no accounts")
	}
}

func TestFindDefaultAccount_NilInfo(t *testing.T) {
	_, err := FindDefaultAccount(nil)
	if err == nil {
		t.Fatal("expected error for nil userinfo")
	}
}
