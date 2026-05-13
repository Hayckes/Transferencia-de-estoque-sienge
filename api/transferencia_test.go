package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"sienge-transfer/models"
)

func TestBuildStockTransferPayloadWithSingleItem(t *testing.T) {
	transfer := validTransferencia()
	transfer.Insumos = transfer.Insumos[:1]

	payload, err := BuildStockTransferPayload(transfer)
	if err != nil {
		t.Fatalf("BuildStockTransferPayload() error = %v", err)
	}

	if payload.SourceCostCenterID != 121 || payload.DestinationCostCenterID != 205 {
		t.Fatalf("payload buildings = %d/%d, want 121/205", payload.SourceCostCenterID, payload.DestinationCostCenterID)
	}
	if payload.DocumentID != "TR" || payload.MovementTypeID != 3 {
		t.Fatalf("payload document/movement = %s/%d, want TR/3", payload.DocumentID, payload.MovementTypeID)
	}
	if payload.MovementDate != "2024-07-15" {
		t.Fatalf("MovementDate = %q, want 2024-07-15", payload.MovementDate)
	}
	if len(payload.Items) != 1 {
		t.Fatalf("len(Items) = %d, want 1", len(payload.Items))
	}
	if payload.Items[0].Source.ResourceID != 3421 || payload.Items[0].Source.DetailID != 10 || payload.Items[0].Source.TrademarkID != 5 || payload.Items[0].Source.Quantity != 50 || payload.Items[0].Source.UnitOfMeasure != "SC" {
		t.Fatalf("Items[0].Source = %#v, want mapped source item", payload.Items[0].Source)
	}
	if payload.Items[0].Destination.ResourceID != 3421 || payload.Items[0].Destination.UnitPrice != 30.5 {
		t.Fatalf("Items[0].Destination = %#v, want mapped destination item", payload.Items[0].Destination)
	}
	if len(payload.Items[0].Source.BuildingAppropriations) != 1 || len(payload.Items[0].Destination.BuildingAppropriations) != 1 {
		t.Fatalf("Items[0] appropriations = %#v, want mapped appropriations", payload.Items[0])
	}
	if !strings.Contains(payload.Notes, "Transferencia realizada por Joao Silva (Engenheiro)") {
		t.Fatalf("Notes = %q, want user and role", payload.Notes)
	}
	if !strings.Contains(payload.Notes, "Observacao: Prioridade alta") {
		t.Fatalf("Notes = %q, want observation", payload.Notes)
	}
}

func TestBuildStockTransferPayloadWithMultipleItems(t *testing.T) {
	payload, err := BuildStockTransferPayload(validTransferencia())
	if err != nil {
		t.Fatalf("BuildStockTransferPayload() error = %v", err)
	}

	if len(payload.Items) != 2 {
		t.Fatalf("len(Items) = %d, want 2", len(payload.Items))
	}
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("json.Marshal(payload) error = %v", err)
	}
	if !strings.Contains(string(data), `"resourceId":3421`) || !strings.Contains(string(data), `"resourceId":9876`) {
		t.Fatalf("payload JSON = %s, want both items", string(data))
	}
}

func TestValidateTransferenciaRejectsInvalidHeaderFields(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*models.Transferencia)
		want   string
	}{
		{name: "missing origin", mutate: func(t *models.Transferencia) { t.ObraOrigemID = 0 }, want: "obra de origem"},
		{name: "missing destination", mutate: func(t *models.Transferencia) { t.ObraDestinoID = 0 }, want: "obra de destino"},
		{name: "same origin and destination", mutate: func(t *models.Transferencia) { t.ObraDestinoID = t.ObraOrigemID }, want: "diferente"},
		{name: "missing requester", mutate: func(t *models.Transferencia) { t.Solicitante = "" }, want: "solicitante"},
		{name: "missing document type", mutate: func(t *models.Transferencia) { t.CodigoTipoDocumento = "" }, want: "documento"},
		{name: "missing movement type", mutate: func(t *models.Transferencia) { t.CodigoTipoMovimento = 0 }, want: "movimento"},
		{name: "missing datetime", mutate: func(t *models.Transferencia) { t.DataHora = time.Time{} }, want: "data e hora"},
		{name: "missing items", mutate: func(t *models.Transferencia) { t.Insumos = nil }, want: "insumo"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transfer := validTransferencia()
			tt.mutate(&transfer)
			errorsList := ValidateTransferencia(transfer)
			if len(errorsList) == 0 {
				t.Fatal("ValidateTransferencia() returned no errors, want at least one")
			}
			if !containsValidationText(errorsList, tt.want) {
				t.Fatalf("ValidateTransferencia() = %#v, want text %q", errorsList, tt.want)
			}
		})
	}
}

func TestValidateTransferenciaRejectsInvalidItems(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*models.ItemTransferido)
		want   string
	}{
		{name: "missing supply id", mutate: func(i *models.ItemTransferido) { i.ID = 0 }, want: "ID do insumo"},
		{name: "missing origin appropriation", mutate: func(i *models.ItemTransferido) { i.Apropriacao = "" }, want: "origem"},
		{name: "missing destination appropriation", mutate: func(i *models.ItemTransferido) { i.ApropriacaoDestino = "" }, want: "destino"},
		{name: "missing origin ids", mutate: func(i *models.ItemTransferido) { i.ApropriacaoOrigemBuildingUnitID = 0 }, want: "identificadores da apropriacao de origem"},
		{name: "missing destination ids", mutate: func(i *models.ItemTransferido) { i.ApropriacaoDestinoBuildingUnitID = 0 }, want: "identificadores da apropriacao de destino"},
		{name: "missing unit", mutate: func(i *models.ItemTransferido) { i.Unidade = "" }, want: "unidade de medida"},
		{name: "missing unit price", mutate: func(i *models.ItemTransferido) { i.PrecoUnitario = 0 }, want: "preco unitario"},
		{name: "zero quantity", mutate: func(i *models.ItemTransferido) { i.Quantidade = 0 }, want: "maior que zero"},
		{name: "negative quantity", mutate: func(i *models.ItemTransferido) { i.Quantidade = -1 }, want: "maior que zero"},
		{name: "quantity greater than available", mutate: func(i *models.ItemTransferido) { i.Quantidade = 11; i.QuantidadeDisponivel = 10 }, want: "maior que a disponivel"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transfer := validTransferencia()
			tt.mutate(&transfer.Insumos[0])
			errorsList := ValidateTransferencia(transfer)
			if len(errorsList) == 0 {
				t.Fatal("ValidateTransferencia() returned no errors, want at least one")
			}
			if !containsValidationText(errorsList, tt.want) {
				t.Fatalf("ValidateTransferencia() = %#v, want text %q", errorsList, tt.want)
			}
		})
	}
}

func TestBuildStockTransferPayloadReturnsValidationError(t *testing.T) {
	transfer := validTransferencia()
	transfer.Solicitante = ""

	_, err := BuildStockTransferPayload(transfer)
	if err == nil {
		t.Fatal("BuildStockTransferPayload() error = nil, want validation error")
	}
	if !IsTransferValidationError(err) {
		t.Fatalf("BuildStockTransferPayload() error type = %T, want TransferValidationError", err)
	}
}

func TestBuildTransferNoteIncludesRequiredContextAndNoSecrets(t *testing.T) {
	note := BuildTransferNote(validTransferencia())

	for _, want := range []string{"Joao Silva", "Engenheiro", "Maria Santos", "Prioridade alta", "121 - Residencial", "205 - Comercial", "3421", "A001", "50"} {
		if !strings.Contains(note, want) {
			t.Fatalf("note = %q, want containing %q", note, want)
		}
	}
	if strings.Contains(strings.ToLower(note), "senha") || strings.Contains(strings.ToLower(note), "token") {
		t.Fatalf("note contains sensitive word: %q", note)
	}
}

func TestCreateStockTransferPostsPayloadAndExtractsIDFromBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if r.URL.String() != "/public/api/v1/stock-movements/transfer" {
			t.Fatalf("URL = %s, want /public/api/v1/stock-movements/transfer", r.URL.String())
		}

		var payload StockTransferPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("Decode(body) error = %v", err)
		}
		if payload.SourceCostCenterID != 121 || payload.Items[0].Source.ResourceID != 3421 {
			t.Fatalf("payload = %#v, want transfer payload", payload)
		}

		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"movementId":"MOV-2024-001"}`))
	}))
	defer server.Close()

	client := newTestClient(t, server.URL+BasePath, nil)
	movementID, err := client.CreateStockTransfer(context.Background(), validTransferencia())
	if err != nil {
		t.Fatalf("CreateStockTransfer() error = %v", err)
	}
	if movementID != "MOV-2024-001" {
		t.Fatalf("movementID = %q, want MOV-2024-001", movementID)
	}
}

func TestCreateStockTransferExtractsIDFromLocationHeader(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", "https://example.com/stock-movements/transfer/7842")
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	client := newTestClient(t, server.URL+BasePath, nil)
	movementID, err := client.CreateStockTransfer(context.Background(), validTransferencia())
	if err != nil {
		t.Fatalf("CreateStockTransfer() error = %v", err)
	}
	if movementID != "7842" {
		t.Fatalf("movementID = %q, want 7842", movementID)
	}
}

func TestCreateStockTransferReturnsEmptyIDWhenNotIdentified(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	client := newTestClient(t, server.URL+BasePath, nil)
	movementID, err := client.CreateStockTransfer(context.Background(), validTransferencia())
	if err != nil {
		t.Fatalf("CreateStockTransfer() error = %v", err)
	}
	if movementID != "" {
		t.Fatalf("movementID = %q, want empty", movementID)
	}
}

func TestCreateStockTransferReturnsAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"field":"items[0].quantity"}`))
	}))
	defer server.Close()

	client := newTestClient(t, server.URL+BasePath, nil)
	_, err := client.CreateStockTransfer(context.Background(), validTransferencia())
	if err == nil {
		t.Fatal("CreateStockTransfer() error = nil, want API error")
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("CreateStockTransfer() error type = %T, want *APIError", err)
	}
	if apiErr.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("StatusCode = %d, want 422", apiErr.StatusCode)
	}
}

func TestCreateStockTransferDryRunDoesNotPost(t *testing.T) {
	t.Setenv(TransferDryRunEnv, "true")
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	client := newTestClient(t, server.URL+BasePath, nil)
	_, err := client.CreateStockTransfer(context.Background(), validTransferencia())
	if err == nil {
		t.Fatal("CreateStockTransfer() error = nil, want dry-run block")
	}
	if calls != 0 {
		t.Fatalf("calls = %d, want no POST in dry-run", calls)
	}
}

func TestCreateStockTransferDoesNotRetryOnHTML(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("<html>blocked</html>"))
	}))
	defer server.Close()

	client := newTestClient(t, server.URL+BasePath, nil)
	_, err := client.CreateStockTransfer(context.Background(), validTransferencia())
	if err == nil {
		t.Fatal("CreateStockTransfer() error = nil, want HTML error")
	}
	if calls != 1 {
		t.Fatalf("calls = %d, want one POST without retry", calls)
	}
}

func TestCreateStockTransferDoesNotRetryOnServerError(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"erro":"falha"}`))
	}))
	defer server.Close()

	client := newTestClient(t, server.URL+BasePath, nil)
	_, err := client.CreateStockTransfer(context.Background(), validTransferencia())
	if err == nil {
		t.Fatal("CreateStockTransfer() error = nil, want server error")
	}
	if calls != 1 {
		t.Fatalf("calls = %d, want one POST without retry", calls)
	}
}

func TestCircuitBreakerBlocksAfterHTMLResponse(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("<html>blocked</html>"))
	}))
	defer server.Close()

	client := newTestClient(t, server.URL+BasePath, nil)
	_, err := client.CreateStockTransfer(context.Background(), validTransferencia())
	if err == nil {
		t.Fatal("first CreateStockTransfer() error = nil, want HTML error")
	}
	_, err = client.CreateStockTransfer(context.Background(), validTransferencia())
	if err == nil {
		t.Fatal("second CreateStockTransfer() error = nil, want circuit breaker error")
	}
	var breakerErr *CircuitBreakerBlockedError
	if !errors.As(err, &breakerErr) {
		t.Fatalf("second error type = %T, want *CircuitBreakerBlockedError", err)
	}
	if calls != 1 {
		t.Fatalf("calls = %d, want second POST blocked before network", calls)
	}
}

func TestCircuitBreakerBlocksAfterRedirect(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		http.Redirect(w, r, "/sienge/internal-api/v1/auth/sso/callback", http.StatusFound)
	}))
	defer server.Close()

	client := newTestClient(t, server.URL+BasePath, nil)
	_, err := client.CreateStockTransfer(context.Background(), validTransferencia())
	if err == nil {
		t.Fatal("first CreateStockTransfer() error = nil, want redirect error")
	}
	_, err = client.CreateStockTransfer(context.Background(), validTransferencia())
	if err == nil {
		t.Fatal("second CreateStockTransfer() error = nil, want circuit breaker error")
	}
	var breakerErr *CircuitBreakerBlockedError
	if !errors.As(err, &breakerErr) {
		t.Fatalf("second error type = %T, want *CircuitBreakerBlockedError", err)
	}
	if calls != 1 {
		t.Fatalf("calls = %d, want redirect not followed and second POST blocked", calls)
	}
}

func TestExtractMovementIDChecksBodyBeforeLocation(t *testing.T) {
	resp := &http.Response{Header: http.Header{"Location": []string{"https://example.com/stock-movements/transfer/7842"}}}
	movementID := ExtractMovementID(resp, []byte(`{"documentNumber":"DOC-1"}`))
	if movementID != "DOC-1" {
		t.Fatalf("ExtractMovementID() = %q, want body ID DOC-1", movementID)
	}
}

func TestExtractMovementIDSupportsKnownBodyFields(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{name: "id", body: `{"id":7842}`, want: "7842"},
		{name: "movementId", body: `{"movementId":"MOV-1"}`, want: "MOV-1"},
		{name: "stockMovementId", body: `{"stockMovementId":"STK-1"}`, want: "STK-1"},
		{name: "documentNumber", body: `{"documentNumber":"DOC-1"}`, want: "DOC-1"},
		{name: "movementNumber", body: `{"movementNumber":"NUM-1"}`, want: "NUM-1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ExtractMovementID(nil, []byte(tt.body)); got != tt.want {
				t.Fatalf("ExtractMovementID() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNewTransferenciaBaseDefaults(t *testing.T) {
	transfer := NewTransferenciaBase()
	if transfer.CodigoTipoDocumento != "TR" || transfer.CodigoTipoMovimento != 3 {
		t.Fatalf("defaults = %s/%d, want TR/3", transfer.CodigoTipoDocumento, transfer.CodigoTipoMovimento)
	}
	if transfer.DataHora.IsZero() {
		t.Fatal("DataHora should be initialized")
	}
}

func containsValidationText(errorsList []string, text string) bool {
	for _, item := range errorsList {
		if strings.Contains(item, text) {
			return true
		}
	}

	return false
}

func validTransferencia() models.Transferencia {
	return models.Transferencia{
		DataHora:            time.Date(2024, 7, 15, 10, 30, 0, 0, time.Local),
		Usuario:             "Joao Silva",
		Cargo:               "Engenheiro",
		Solicitante:         "Maria Santos",
		Observacao:          "Prioridade alta",
		ObraOrigemID:        121,
		ObraOrigemNome:      "Residencial Novo Horizonte",
		ObraDestinoID:       205,
		ObraDestinoNome:     "Comercial Centro",
		CodigoTipoDocumento: "TR",
		CodigoTipoMovimento: 3,
		Insumos: []models.ItemTransferido{
			{
				ID:                               3421,
				Nome:                             "Cimento",
				Detalhe:                          "CP III",
				DetalheID:                        10,
				Marca:                            "Votorantim",
				MarcaID:                          5,
				Unidade:                          "SC",
				PrecoUnitario:                    30.5,
				Apropriacao:                      "A001",
				ApropriacaoDescricao:             "Fundacao",
				ApropriacaoOrigemBuildingUnitID:  3,
				ApropriacaoOrigemSheetItemID:     4,
				ApropriacaoDestino:               "D001",
				ApropriacaoDestinoDescricao:      "Destino",
				ApropriacaoDestinoBuildingUnitID: 7,
				ApropriacaoDestinoSheetItemID:    8,
				ApropriacaoOrigemObrigatoria:     true,
				ApropriacaoDestinoObrigatoria:    true,
				Quantidade:                       50,
				QuantidadeDisponivel:             150,
			},
			{
				ID:                               9876,
				Nome:                             "Areia",
				Detalhe:                          "Media",
				DetalheID:                        11,
				Marca:                            "Regional",
				MarcaID:                          6,
				Unidade:                          "M3",
				PrecoUnitario:                    12.25,
				Apropriacao:                      "A002",
				ApropriacaoDescricao:             "Estrutura",
				ApropriacaoOrigemBuildingUnitID:  9,
				ApropriacaoOrigemSheetItemID:     10,
				ApropriacaoDestino:               "D002",
				ApropriacaoDestinoDescricao:      "Acabamento",
				ApropriacaoDestinoBuildingUnitID: 11,
				ApropriacaoDestinoSheetItemID:    12,
				ApropriacaoOrigemObrigatoria:     true,
				ApropriacaoDestinoObrigatoria:    true,
				Quantidade:                       20.5,
				QuantidadeDisponivel:             30,
			},
		},
	}
}
