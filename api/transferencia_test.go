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

	if payload.OriginBuildingID != 121 || payload.DestinationBuildingID != 205 {
		t.Fatalf("payload buildings = %d/%d, want 121/205", payload.OriginBuildingID, payload.DestinationBuildingID)
	}
	if payload.DocumentTypeCode != "TR" || payload.MovementTypeCode != 3 {
		t.Fatalf("payload document/movement = %s/%d, want TR/3", payload.DocumentTypeCode, payload.MovementTypeCode)
	}
	if payload.TransferDate != "2024-07-15T10:30:00" {
		t.Fatalf("TransferDate = %q, want 2024-07-15T10:30:00", payload.TransferDate)
	}
	if len(payload.Items) != 1 {
		t.Fatalf("len(Items) = %d, want 1", len(payload.Items))
	}
	if payload.Items[0].SupplyID != 3421 || payload.Items[0].Detail != "CP III" || payload.Items[0].Brand != "Votorantim" || payload.Items[0].AppropriationCode != "A001" || payload.Items[0].Quantity != 50 {
		t.Fatalf("Items[0] = %#v, want mapped item", payload.Items[0])
	}
	if !strings.Contains(payload.Note, "Transferencia realizada por Joao Silva (Engenheiro)") {
		t.Fatalf("Note = %q, want user and role", payload.Note)
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
	if !strings.Contains(string(data), `"supplyId":3421`) || !strings.Contains(string(data), `"supplyId":9876`) {
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
		{name: "missing appropriation", mutate: func(i *models.ItemTransferido) { i.Apropriacao = "" }, want: "apropriacao"},
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

	for _, want := range []string{"Joao Silva", "Engenheiro", "Maria Santos", "121 - Residencial", "205 - Comercial", "3421", "A001", "50"} {
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
		if r.URL.String() != "/sienge/api/public/v1/stock-transfers" {
			t.Fatalf("URL = %s, want /sienge/api/public/v1/stock-transfers", r.URL.String())
		}

		var payload StockTransferPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("Decode(body) error = %v", err)
		}
		if payload.OriginBuildingID != 121 || payload.Items[0].SupplyID != 3421 {
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
		w.Header().Set("Location", "https://example.com/stock-transfers/7842")
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

func TestExtractMovementIDChecksBodyBeforeLocation(t *testing.T) {
	resp := &http.Response{Header: http.Header{"Location": []string{"https://example.com/stock-transfers/7842"}}}
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
		ObraOrigemID:        121,
		ObraOrigemNome:      "Residencial Novo Horizonte",
		ObraDestinoID:       205,
		ObraDestinoNome:     "Comercial Centro",
		CodigoTipoDocumento: "TR",
		CodigoTipoMovimento: 3,
		Insumos: []models.ItemTransferido{
			{
				ID:                   3421,
				Nome:                 "Cimento",
				Detalhe:              "CP III",
				Marca:                "Votorantim",
				Apropriacao:          "A001",
				ApropriacaoDescricao: "Fundacao",
				Quantidade:           50,
				QuantidadeDisponivel: 150,
			},
			{
				ID:                   9876,
				Nome:                 "Areia",
				Detalhe:              "Media",
				Marca:                "Regional",
				Apropriacao:          "A002",
				ApropriacaoDescricao: "Estrutura",
				Quantidade:           20.5,
				QuantidadeDisponivel: 30,
			},
		},
	}
}
