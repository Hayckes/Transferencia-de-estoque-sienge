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
	"regexp"
	"strings"
	"sync"
	"time"
)

const (
	BasePath              = "/public/api/v1"
	DefaultTimeout        = 30 * time.Second
	maxErrorBody          = 4096
	transferBlockDuration = 10 * time.Minute
	transferPostInterval  = 30 * time.Second
)

const (
	APIErrorKindHTML     = "html"
	APIErrorKindRedirect = "redirect"
	APIErrorKindTimeout  = "timeout"
)

var validSubdomainPattern = regexp.MustCompile(`^[a-zA-Z0-9-]+$`)

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
	Kind       string
}

type CircuitBreakerState struct {
	Tenant        string
	BlockedUntil  time.Time
	Reason        string
	LastRequestID string
}

type CircuitBreaker struct {
	mu       sync.Mutex
	duration time.Duration
	now      func() time.Time
	states   map[string]CircuitBreakerState
}

type CircuitBreakerBlockedError struct {
	State CircuitBreakerState
}

func (e *CircuitBreakerBlockedError) Error() string {
	return fmt.Sprintf("Envio de transferencias bloqueado temporariamente por seguranca. Motivo: %s. Aguarde ate %s ou revise a configuracao da API antes de tentar novamente.", e.State.Reason, e.State.BlockedUntil.Format("15:04:05"))
}

type transferPostGate struct {
	mu          sync.Mutex
	inFlight    map[string]bool
	lastStarted map[string]time.Time
	now         func() time.Time
}

var stockTransferCircuitBreaker = NewCircuitBreaker(transferBlockDuration)
var stockTransferPostGate = newTransferPostGate()

type apiResponse struct {
	StatusCode int
	Header     http.Header
	Body       []byte
}

func (e *APIError) Error() string {
	if e.Body == "" {
		return e.Message
	}

	return fmt.Sprintf("%s: %s", e.Message, e.Body)
}

func NewClient(subdomain, username, password string) (*Client, error) {
	apiURL, err := NewSiengeAPIBaseURL(subdomain)
	if err != nil {
		return nil, err
	}
	baseURL := strings.TrimRight(apiURL.String(), "/")
	return NewClientWithBaseURL(baseURL, username, password, nil)
}

func NewSiengeAPIBaseURL(subdomain string) (*url.URL, error) {
	subdomain = strings.TrimSpace(subdomain)
	if subdomain == "" {
		return nil, errors.New("subdominio da empresa obrigatorio")
	}
	lower := strings.ToLower(subdomain)
	if strings.Contains(lower, "://") || strings.ContainsAny(subdomain, "/?#") {
		return nil, errors.New("subdominio deve conter apenas o identificador da empresa, sem URL ou caminho")
	}
	for _, forbidden := range []string{"internal-api", "callback", "sienge/api"} {
		if strings.Contains(lower, forbidden) {
			return nil, errors.New("subdominio contem caminho de API/web nao permitido")
		}
	}
	if !validSubdomainPattern.MatchString(subdomain) {
		return nil, errors.New("subdominio contem caracteres invalidos")
	}

	return url.Parse(fmt.Sprintf("https://api.sienge.com.br/%s%s", url.PathEscape(subdomain), BasePath))
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
		httpClient = defaultHTTPClient()
	} else {
		copyClient := *httpClient
		if copyClient.CheckRedirect == nil {
			copyClient.CheckRedirect = dontFollowRedirects
		}
		httpClient = &copyClient
	}

	return &Client{
		baseURL:    baseURL,
		username:   username,
		password:   password,
		httpClient: httpClient,
	}, nil
}

func defaultHTTPClient() *http.Client {
	return &http.Client{
		Timeout:       DefaultTimeout,
		CheckRedirect: dontFollowRedirects,
		Transport: &http.Transport{
			MaxIdleConns:        10,
			MaxIdleConnsPerHost: 2,
			IdleConnTimeout:     30 * time.Second,
		},
	}
}

func dontFollowRedirects(req *http.Request, via []*http.Request) error {
	return http.ErrUseLastResponse
}

func (c *Client) BaseURL() string {
	return c.baseURL
}

func (c *Client) ValidateCredentials(ctx context.Context) error {
	_, err := c.GetCostCenters(ctx, 1)
	if errors.Is(err, ErrCostCenterNotFound) {
		return nil
	}
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
	resp, err := c.doResponse(ctx, method, path, body)
	if err != nil {
		return nil, err
	}

	return resp.Body, nil
}

func (c *Client) doResponse(ctx context.Context, method, path string, body []byte) (*apiResponse, error) {
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
			return nil, &APIError{Message: "Tempo limite excedido ao comunicar com o Sienge. Nao reenvie a transferencia sem consultar o Sienge antes.", Kind: APIErrorKindTimeout}
		}
		return nil, fmt.Errorf("falha de comunicacao com o Sienge: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if isRedirectStatus(resp.StatusCode) {
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Message:    "O Sienge redirecionou a chamada da API para outro endereco. Isso indica URL base incorreta, credencial invalida ou endpoint indisponivel para esta empresa.",
			Body:       sanitizeRedirectLocation(resp.Header.Get("Location")),
			Kind:       APIErrorKindRedirect,
		}
	}
	if isHTMLResponse(resp.Header, respBody) {
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Message:    "O Sienge retornou resposta em formato HTML em uma chamada de API. Isso normalmente indica redirecionamento, bloqueio temporario, URL incorreta ou endpoint indisponivel.",
			Kind:       APIErrorKindHTML,
		}
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		if len(respBody) > maxErrorBody {
			respBody = respBody[:maxErrorBody]
		}
		return nil, newAPIError(resp.StatusCode, respBody)
	}

	return &apiResponse{
		StatusCode: resp.StatusCode,
		Header:     resp.Header.Clone(),
		Body:       respBody,
	}, nil
}

func isHTMLResponse(header http.Header, body []byte) bool {
	contentType := strings.ToLower(header.Get("Content-Type"))
	if strings.Contains(contentType, "text/html") {
		return true
	}

	trimmed := strings.ToLower(string(bytes.TrimSpace(body)))
	return strings.HasPrefix(trimmed, "<html") || strings.HasPrefix(trimmed, "<!doctype html")
}

func isRedirectStatus(statusCode int) bool {
	switch statusCode {
	case http.StatusMovedPermanently, http.StatusFound, http.StatusSeeOther, http.StatusTemporaryRedirect, http.StatusPermanentRedirect:
		return true
	default:
		return false
	}
}

func sanitizeRedirectLocation(location string) string {
	location = strings.TrimSpace(location)
	if location == "" {
		return ""
	}
	parsed, err := url.Parse(location)
	if err != nil {
		return redactSensitiveWords(location)
	}
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return redactSensitiveWords(parsed.String())
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
	message := messageForStatus(statusCode)
	clientMessage := extractClientMessage(body)
	if clientMessage != "" {
		message = messageWithClientDetail(statusCode, clientMessage)
	}
	bodyText := sanitizeBody(string(body))
	if clientMessage != "" {
		bodyText = ""
	}

	return &APIError{
		StatusCode: statusCode,
		Message:    message,
		Body:       bodyText,
	}
}

func messageForStatus(statusCode int) string {
	switch {
	case isRedirectStatus(statusCode):
		return "O Sienge redirecionou a chamada da API para outro endereco. Verifique a URL base e as credenciais."
	case statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden:
		return "Credenciais invalidas ou sem permissao. Refaca o onboarding das credenciais da API."
	case statusCode == http.StatusTooManyRequests:
		return "Limite de chamadas do Sienge atingido. Aguarde antes de tentar novamente."
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

func NewCircuitBreaker(duration time.Duration) *CircuitBreaker {
	return &CircuitBreaker{duration: duration, now: time.Now, states: make(map[string]CircuitBreakerState)}
}

func (b *CircuitBreaker) Check(tenant string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	state, ok := b.states[tenant]
	if !ok {
		return nil
	}
	if !b.now().Before(state.BlockedUntil) {
		delete(b.states, tenant)
		return nil
	}
	return &CircuitBreakerBlockedError{State: state}
}

func (b *CircuitBreaker) Block(tenant, reason, requestID string) CircuitBreakerState {
	b.mu.Lock()
	defer b.mu.Unlock()

	state := CircuitBreakerState{Tenant: tenant, BlockedUntil: b.now().Add(b.duration), Reason: strings.TrimSpace(reason), LastRequestID: requestID}
	if state.Reason == "" {
		state.Reason = "resposta anormal do Sienge"
	}
	b.states[tenant] = state
	return state
}

func (b *CircuitBreaker) Reset() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.states = make(map[string]CircuitBreakerState)
}

func newTransferPostGate() *transferPostGate {
	return &transferPostGate{inFlight: make(map[string]bool), lastStarted: make(map[string]time.Time), now: time.Now}
}

func (g *transferPostGate) Begin(tenant string) (func(), error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.inFlight[tenant] {
		return nil, errors.New("Transferencia ja esta em envio para esta empresa. Aguarde a conclusao.")
	}
	now := g.now()
	if last, ok := g.lastStarted[tenant]; ok && now.Sub(last) < transferPostInterval {
		return nil, fmt.Errorf("Aguarde %s antes de enviar outra transferencia para esta empresa.", (transferPostInterval - now.Sub(last)).Round(time.Second))
	}
	g.inFlight[tenant] = true
	g.lastStarted[tenant] = now

	return func() {
		g.mu.Lock()
		defer g.mu.Unlock()
		delete(g.inFlight, tenant)
	}, nil
}

func (g *transferPostGate) Reset() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.inFlight = make(map[string]bool)
	g.lastStarted = make(map[string]time.Time)
}

func resetTransferSafetyStateForTests() {
	stockTransferCircuitBreaker.Reset()
	stockTransferPostGate.Reset()
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

func extractClientMessage(body []byte) string {
	var data map[string]any
	if json.Unmarshal(body, &data) != nil {
		return ""
	}

	for _, key := range []string{"clientMessage", "userMessage"} {
		if value, ok := data[key]; ok && value != nil {
			text := strings.TrimSpace(fmt.Sprint(value))
			if text != "" {
				return text
			}
		}
	}

	return ""
}

func messageWithClientDetail(statusCode int, detail string) string {
	detail = strings.TrimRight(strings.TrimSpace(detail), ".")
	if statusCode == http.StatusUnprocessableEntity {
		message := "O Sienge recusou a solicitacao: " + detail
		if strings.Contains(strings.ToLower(detail), "bloquead") {
			message += ". Selecione outra apropriacao ou desbloqueie o item no Sienge."
		}
		return message
	}

	return messageForStatus(statusCode) + ": " + detail
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
