package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	BasePath       = "/sienge/api/public/v1"
	DefaultTimeout = 30 * time.Second
	maxErrorBody   = 4096
)

type Client struct {
	baseURL    string
	username   string
	password   string
	httpClient *http.Client
}

type APIError struct {
	StatusCode int
	Message    string
	Body       string
}

func (e *APIError) Error() string {
	if e.Body == "" {
		return e.Message
	}

	return fmt.Sprintf("%s: %s", e.Message, e.Body)
}

func NewClient(subdomain, username, password string) (*Client, error) {
	subdomain = strings.TrimSpace(subdomain)
	if subdomain == "" {
		return nil, errors.New("subdominio da empresa obrigatorio")
	}

	baseURL := fmt.Sprintf("https://%s.sienge.com.br%s", subdomain, BasePath)
	return NewClientWithBaseURL(baseURL, username, password, nil)
}

func NewClientWithBaseURL(baseURL, username, password string, httpClient *http.Client) (*Client, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return nil, errors.New("URL base da API obrigatoria")
	}
	if _, err := url.ParseRequestURI(baseURL); err != nil {
		return nil, fmt.Errorf("URL base da API invalida: %w", err)
	}
	if strings.TrimSpace(username) == "" {
		return nil, errors.New("usuario da API obrigatorio")
	}
	if strings.TrimSpace(password) == "" {
		return nil, errors.New("senha da API obrigatoria")
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: DefaultTimeout}
	}

	return &Client{
		baseURL:    baseURL,
		username:   username,
		password:   password,
		httpClient: httpClient,
	}, nil
}

func (c *Client) BaseURL() string {
	return c.baseURL
}

func (c *Client) ValidateCredentials(ctx context.Context) error {
	_, err := c.do(ctx, http.MethodGet, "/buildings?limit=1", nil)
	return err
}

func (c *Client) PostJSON(ctx context.Context, path string, payload any) ([]byte, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	return c.do(ctx, http.MethodPost, path, body)
}

func (c *Client) do(ctx context.Context, method, path string, body []byte) ([]byte, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	endpoint, err := c.endpoint(path)
	if err != nil {
		return nil, err
	}

	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, endpoint, reader)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(c.username, c.password)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return nil, &APIError{Message: "Tempo limite excedido ao comunicar com o Sienge. Tente novamente."}
		}
		return nil, fmt.Errorf("falha de comunicacao com o Sienge: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxErrorBody+1))
	if err != nil {
		return nil, err
	}
	if len(respBody) > maxErrorBody {
		respBody = respBody[:maxErrorBody]
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, newAPIError(resp.StatusCode, respBody)
	}

	return respBody, nil
}

func (c *Client) endpoint(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", errors.New("endpoint da API obrigatorio")
	}
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return "", errors.New("endpoint deve ser relativo a URL base da API")
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	return c.baseURL + path, nil
}

func newAPIError(statusCode int, body []byte) *APIError {
	return &APIError{
		StatusCode: statusCode,
		Message:    messageForStatus(statusCode),
		Body:       sanitizeBody(string(body)),
	}
}

func messageForStatus(statusCode int) string {
	switch {
	case statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden:
		return "Credenciais invalidas ou sem permissao. Refaca o onboarding das credenciais da API."
	case statusCode == http.StatusNotFound:
		return "Recurso nao encontrado no Sienge. Verifique se a obra, insumo ou endpoint esta correto."
	case statusCode == http.StatusUnprocessableEntity:
		return "Dados invalidos enviados ao Sienge. Revise os campos informados."
	case statusCode >= 500:
		return "Erro no servidor do Sienge. Tente novamente em alguns instantes."
	default:
		return fmt.Sprintf("Erro ao comunicar com o Sienge. Codigo HTTP: %d", statusCode)
	}
}

func sanitizeBody(body string) string {
	body = strings.TrimSpace(body)
	if body == "" {
		return ""
	}

	var data any
	if json.Unmarshal([]byte(body), &data) == nil {
		data = sanitizeJSONValue(data)
		encoded, err := json.Marshal(data)
		if err == nil {
			return string(encoded)
		}
	}

	return redactSensitiveWords(body)
}

func sanitizeJSONValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		clean := make(map[string]any, len(typed))
		for key, val := range typed {
			if isSensitiveKey(key) {
				clean[key] = "[removido]"
				continue
			}
			clean[key] = sanitizeJSONValue(val)
		}
		return clean
	case []any:
		clean := make([]any, len(typed))
		for i, val := range typed {
			clean[i] = sanitizeJSONValue(val)
		}
		return clean
	default:
		return value
	}
}

func isSensitiveKey(key string) bool {
	key = strings.ToLower(key)
	return strings.Contains(key, "senha") || strings.Contains(key, "password") || strings.Contains(key, "token") || strings.Contains(key, "authorization")
}

func redactSensitiveWords(body string) string {
	words := []string{"password", "senha", "token", "authorization"}
	redacted := body
	for _, word := range words {
		redacted = strings.ReplaceAll(redacted, word, "[removido]")
		redacted = strings.ReplaceAll(redacted, strings.ToUpper(word), "[removido]")
	}

	return redacted
}
