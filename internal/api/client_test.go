package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClient_Get(t *testing.T) {
	type response struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}

		if r.URL.Path != "/test/resource" {
			t.Errorf("expected path /test/resource, got %s", r.URL.Path)
		}

		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("expected Authorization header 'Bearer test-token', got %q", r.Header.Get("Authorization"))
		}

		if r.Header.Get("User-Agent") != "docusign-cli/1.0" {
			t.Errorf("expected User-Agent 'docusign-cli/1.0', got %q", r.Header.Get("User-Agent"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response{ID: "123", Name: "test"})
	}))
	defer server.Close()

	client := NewClient("test-token",
		WithBaseURL(server.URL),
		WithUserAgent("docusign-cli/1.0"),
	)

	var result response

	err := client.Get(context.Background(), "/test/resource", &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ID != "123" {
		t.Errorf("expected ID '123', got %q", result.ID)
	}

	if result.Name != "test" {
		t.Errorf("expected Name 'test', got %q", result.Name)
	}
}

func TestClient_Post(t *testing.T) {
	type requestBody struct {
		Subject string `json:"subject"`
	}

	type response struct {
		ID string `json:"id"`
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type 'application/json', got %q", r.Header.Get("Content-Type"))
		}

		body, _ := io.ReadAll(r.Body)

		var req requestBody
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("unmarshal request: %v", err)
		}

		if req.Subject != "test" {
			t.Errorf("expected subject 'test', got %q", req.Subject)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response{ID: "new-123"})
	}))
	defer server.Close()

	client := NewClient("test-token", WithBaseURL(server.URL))

	req := requestBody{Subject: "test"}

	var result response

	err := client.Post(context.Background(), "/create", req, &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ID != "new-123" {
		t.Errorf("expected ID 'new-123', got %q", result.ID)
	}
}

func TestClient_Delete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewClient("test-token", WithBaseURL(server.URL))

	err := client.Delete(context.Background(), "/resource/123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClient_APIError_401(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Invalid token",
		})
	}))
	defer server.Close()

	client := NewClient("bad-token", WithBaseURL(server.URL))

	var result map[string]any

	err := client.Get(context.Background(), "/test", &result)
	if err == nil {
		t.Fatal("expected error for 401 response")
	}

	var apiErr *APIError

	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T: %v", err, err)
	}

	if apiErr.StatusCode != 401 {
		t.Errorf("expected status 401, got %d", apiErr.StatusCode)
	}

	if apiErr.Message != "Invalid token" {
		t.Errorf("expected message 'Invalid token', got %q", apiErr.Message)
	}
}

func TestClient_APIError_500_NoJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	client := NewClient("test-token", WithBaseURL(server.URL))

	var result map[string]any

	err := client.Get(context.Background(), "/test", &result)
	if err == nil {
		t.Fatal("expected error for 500 response")
	}

	var apiErr *APIError

	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T", err)
	}

	if apiErr.StatusCode != 500 {
		t.Errorf("expected status 500, got %d", apiErr.StatusCode)
	}

	if apiErr.Message != "Internal Server Error" {
		t.Errorf("expected 'Internal Server Error', got %q", apiErr.Message)
	}
}

func TestClient_Put(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
	}))
	defer server.Close()

	client := NewClient("test-token", WithBaseURL(server.URL))

	var result map[string]string

	err := client.Put(context.Background(), "/resource/123", map[string]string{"name": "new"}, &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result["status"] != "updated" {
		t.Errorf("expected status 'updated', got %q", result["status"])
	}
}

func TestClient_CustomHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Custom") != "value" {
			t.Errorf("expected X-Custom header 'value', got %q", r.Header.Get("X-Custom"))
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{})
	}))
	defer server.Close()

	client := NewClient("test-token", WithBaseURL(server.URL))

	resp, err := client.Do(context.Background(), Request{
		Method:  http.MethodGet,
		Path:    "/test",
		Headers: map[string]string{"X-Custom": "value"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	resp.Body.Close()
}

func TestAPIError_Error(t *testing.T) {
	err := &APIError{StatusCode: 404, Message: "Not Found"}
	expected := "API error (404): Not Found"

	if err.Error() != expected {
		t.Errorf("Error() = %q, want %q", err.Error(), expected)
	}
}

func TestWithTimeout(t *testing.T) {
	client := NewClient("key")
	WithTimeout(5000000000)(client) // 5 seconds

	if client.httpClient.Timeout != 5000000000 {
		t.Errorf("expected timeout 5s, got %v", client.httpClient.Timeout)
	}
}
