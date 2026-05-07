package api

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewClientBuildsBaseURLAndTimeout(t *testing.T) {
	client, err := NewClient("minhaempresa", "usuario", "senha")
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	wantBaseURL := "https://api.sienge.com.br/minhaempresa/public/api/v1"
	if client.BaseURL() != wantBaseURL {
		t.Fatalf("BaseURL() = %q, want %q", client.BaseURL(), wantBaseURL)
	}
	if client.httpClient.Timeout != DefaultTimeout {
		t.Fatalf("Timeout = %v, want %v", client.httpClient.Timeout, DefaultTimeout)
	}
}

func TestNewClientWithBaseURLValidatesRequiredFields(t *testing.T) {
	tests := []struct {
		name     string
		baseURL  string
		username string
		password string
	}{
		{name: "missing base url", baseURL: "", username: "usuario", password: "senha"},
		{name: "invalid base url", baseURL: "://invalid", username: "usuario", password: "senha"},
		{name: "missing user", baseURL: "https://example.com", username: "", password: "senha"},
		{name: "missing password", baseURL: "https://example.com", username: "usuario", password: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewClientWithBaseURL(tt.baseURL, tt.username, tt.password, nil)
			if err == nil {
				t.Fatal("NewClientWithBaseURL() error = nil, want error")
			}
		})
	}
}

func TestValidateCredentialsUsesBuildingsEndpointAndBasicAuth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		if r.URL.String() != "/public/api/v1/buildings?limit=1" {
			t.Fatalf("URL = %s, want /public/api/v1/buildings?limit=1", r.URL.String())
		}

		wantAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte("usuario:senha"))
		if r.Header.Get("Authorization") != wantAuth {
			t.Fatalf("Authorization = %q, want %q", r.Header.Get("Authorization"), wantAuth)
		}
		if r.Header.Get("Accept") != "application/json" {
			t.Fatalf("Accept = %q, want application/json", r.Header.Get("Accept"))
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"results":[]}`))
	}))
	defer server.Close()

	client := newTestClient(t, server.URL+BasePath, nil)
	if err := client.ValidateCredentials(context.Background()); err != nil {
		t.Fatalf("ValidateCredentials() error = %v", err)
	}
}

func TestValidateCredentialsMapsHTTPStatuses(t *testing.T) {
	tests := []struct {
		statusCode int
		wantText   string
	}{
		{statusCode: http.StatusUnauthorized, wantText: "Credenciais invalidas"},
		{statusCode: http.StatusForbidden, wantText: "Credenciais invalidas"},
		{statusCode: http.StatusNotFound, wantText: "Recurso nao encontrado"},
		{statusCode: http.StatusUnprocessableEntity, wantText: "Dados invalidos"},
		{statusCode: http.StatusInternalServerError, wantText: "Erro no servidor"},
	}

	for _, tt := range tests {
		t.Run(http.StatusText(tt.statusCode), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(`{"erro":"falha"}`))
			}))
			defer server.Close()

			client := newTestClient(t, server.URL+BasePath, nil)
			err := client.ValidateCredentials(context.Background())
			if err == nil {
				t.Fatal("ValidateCredentials() error = nil, want error")
			}

			var apiErr *APIError
			if !errors.As(err, &apiErr) {
				t.Fatalf("ValidateCredentials() error type = %T, want *APIError", err)
			}
			if apiErr.StatusCode != tt.statusCode {
				t.Fatalf("StatusCode = %d, want %d", apiErr.StatusCode, tt.statusCode)
			}
			if !strings.Contains(apiErr.Message, tt.wantText) {
				t.Fatalf("Message = %q, want containing %q", apiErr.Message, tt.wantText)
			}
		})
	}
}

func TestValidateCredentialsTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := newTestClient(t, server.URL+BasePath, &http.Client{Timeout: 1 * time.Millisecond})
	err := client.ValidateCredentials(context.Background())
	if err == nil {
		t.Fatal("ValidateCredentials() error = nil, want timeout error")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("ValidateCredentials() error type = %T, want *APIError", err)
	}
	if !strings.Contains(apiErr.Message, "Tempo limite excedido") {
		t.Fatalf("Message = %q, want timeout message", apiErr.Message)
	}
}

func TestPostJSONSendsJSONPayloadAndHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if r.URL.String() != "/public/api/v1/stock-transfers" {
			t.Fatalf("URL = %s, want /public/api/v1/stock-transfers", r.URL.String())
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Fatalf("Content-Type = %q, want application/json", r.Header.Get("Content-Type"))
		}

		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("Decode(body) error = %v", err)
		}
		if payload["documentTypeCode"] != "TR" {
			t.Fatalf("documentTypeCode = %v, want TR", payload["documentTypeCode"])
		}

		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"7842"}`))
	}))
	defer server.Close()

	client := newTestClient(t, server.URL+BasePath, nil)
	body, err := client.PostJSON(context.Background(), "/stock-transfers", map[string]any{"documentTypeCode": "TR"})
	if err != nil {
		t.Fatalf("PostJSON() error = %v", err)
	}
	if string(body) != `{"id":"7842"}` {
		t.Fatalf("PostJSON() body = %s, want response JSON", string(body))
	}
}

func TestAPIErrorSanitizesSensitiveJSONBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"message":"erro","password":"senha-secreta","nested":{"token":"abc"}}`))
	}))
	defer server.Close()

	client := newTestClient(t, server.URL+BasePath, nil)
	_, err := client.PostJSON(context.Background(), "/stock-transfers", map[string]any{"ok": true})
	if err == nil {
		t.Fatal("PostJSON() error = nil, want error")
	}

	if strings.Contains(err.Error(), "senha-secreta") || strings.Contains(err.Error(), "abc") {
		t.Fatalf("error leaked sensitive data: %v", err)
	}
	if !strings.Contains(err.Error(), "[removido]") {
		t.Fatalf("error = %v, want redacted marker", err)
	}
}

func TestPostJSONRejectsAbsoluteEndpoint(t *testing.T) {
	client := newTestClient(t, "https://example.com"+BasePath, nil)
	_, err := client.PostJSON(context.Background(), "https://example.com/stock-transfers", map[string]any{})
	if err == nil {
		t.Fatal("PostJSON() error = nil, want error")
	}
}

func TestDoResponseRejectsHTMLSuccessResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte("<html>login</html>"))
	}))
	defer server.Close()

	client := newTestClient(t, server.URL+BasePath, nil)
	err := client.ValidateCredentials(context.Background())
	if err == nil {
		t.Fatal("ValidateCredentials() error = nil, want HTML response error")
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("ValidateCredentials() error type = %T, want *APIError", err)
	}
	if !strings.Contains(apiErr.Message, "formato HTML") {
		t.Fatalf("Message = %q, want HTML format message", apiErr.Message)
	}
}

func newTestClient(t *testing.T, baseURL string, httpClient *http.Client) *Client {
	t.Helper()

	client, err := NewClientWithBaseURL(baseURL, "usuario", "senha", httpClient)
	if err != nil {
		t.Fatalf("NewClientWithBaseURL() error = %v", err)
	}

	return client
}
