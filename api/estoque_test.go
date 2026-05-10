package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"sienge-transfer/models"
)

func TestGetStockItemsCallsCurrentEndpointAndParsesItems(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		if r.URL.String() != "/public/api/v1/stock-inventories/121/items" {
			t.Fatalf("URL = %s, want stock inventory items endpoint", r.URL.String())
		}

		_, _ = w.Write([]byte(`{
			"results": [
				{
					"resourceId": 3421,
					"resourceName": "Cimento",
					"detailDescription": "CP III",
					"trademarkDescription": "Votorantim",
					"quantity": 150,
					"unitOfMeasure": "SC"
				},
				{
					"resourceId": 3421,
					"resourceName": "Cimento",
					"detailDescription": "CP II",
					"trademarkDescription": "Intercement",
					"quantity": 80,
					"unitOfMeasure": "SC"
				}
			]
		}`))
	}))
	defer server.Close()

	client := newTestClient(t, server.URL+BasePath, nil)
	items, err := client.GetStockItems(context.Background(), 121)
	if err != nil {
		t.Fatalf("GetStockItems() error = %v", err)
	}

	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(items))
	}
	if items[0].ID != 3421 || items[0].Nome != "Cimento" || items[0].Detalhe != "CP III" || items[0].Marca != "Votorantim" || items[0].Quantidade != 150 || items[0].Unidade != "SC" {
		t.Fatalf("items[0] = %#v, want parsed stock item", items[0])
	}
	if items[1].ID != 3421 || items[1].Detalhe != "CP II" || items[1].Marca != "Intercement" {
		t.Fatalf("items[1] = %#v, want same ID with different detail/brand", items[1])
	}
	if items[0].OriginalJSON == "" {
		t.Fatal("OriginalJSON should keep raw item data")
	}
}

func TestGetStockItemsByIDsRequiresIDsBeforeCallingAPI(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := newTestClient(t, server.URL+BasePath, nil)
	_, err := client.GetStockItemsByIDs(context.Background(), 121, nil)
	if !errors.Is(err, models.ErrIDsInsumoObrigatorios) {
		t.Fatalf("GetStockItemsByIDs() error = %v, want ErrIDsInsumoObrigatorios", err)
	}
	if called {
		t.Fatal("API should not be called when supply IDs are empty")
	}
}

func TestGetStockItemsByIDsFiltersLocally(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{
			"results": [
				{"resourceId": 3421, "resourceName": "Cimento", "quantity": 10},
				{"resourceId": 9999, "resourceName": "Areia", "quantity": 20},
				{"resourceId": 3421, "resourceName": "Cimento", "detailDescription": "CP II", "quantity": 30}
			]
		}`))
	}))
	defer server.Close()

	client := newTestClient(t, server.URL+BasePath, nil)
	items, err := client.GetStockItemsByIDs(context.Background(), 121, []int{3421})
	if err != nil {
		t.Fatalf("GetStockItemsByIDs() error = %v", err)
	}

	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(items))
	}
	for _, item := range items {
		if item.ID != 3421 {
			t.Fatalf("filtered item ID = %d, want 3421", item.ID)
		}
	}
}

func TestGetStockItemsParsesSuccessfulResponseLargerThanErrorLimit(t *testing.T) {
	largeName := strings.Repeat("Cimento", 700)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"results":[{"resourceId":3421,"resourceName":"` + largeName + `","quantity":10}]}`))
	}))
	defer server.Close()

	client := newTestClient(t, server.URL+BasePath, nil)
	items, err := client.GetStockItems(context.Background(), 121)
	if err != nil {
		t.Fatalf("GetStockItems() error = %v", err)
	}
	if len(items) != 1 || items[0].Nome != largeName {
		t.Fatalf("items = %#v, want untruncated large response", items)
	}
}

func TestGetStockItemsByIDsRejectsInvalidIDs(t *testing.T) {
	client := newTestClient(t, "https://example.com"+BasePath, nil)

	_, err := client.GetStockItemsByIDs(context.Background(), 121, []int{0})
	if err == nil {
		t.Fatal("GetStockItemsByIDs() error = nil, want error")
	}
}

func TestGetStockItemsRejectsInvalidCostCenter(t *testing.T) {
	client := newTestClient(t, "https://example.com"+BasePath, nil)

	_, err := client.GetStockItems(context.Background(), 0)
	if !errors.Is(err, ErrInvalidCostCenter) {
		t.Fatalf("GetStockItems() error = %v, want ErrInvalidCostCenter", err)
	}
}

func TestGetBuildingAppropriationsCallsEndpointAndParsesItems(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.String() != "/public/api/v1/stock-inventories/121/items/3421/building-appropriation" {
			t.Fatalf("URL = %s, want appropriation endpoint", r.URL.String())
		}

		_, _ = w.Write([]byte(`{
			"results": [
				{"appropriationCode": "A001", "appropriationDescription": "Fundacao", "availableQuantity": 40.5},
				{"appropriationCode": "A002", "appropriationDescription": "Estrutura", "availableQuantity": "15,5"}
			]
		}`))
	}))
	defer server.Close()

	client := newTestClient(t, server.URL+BasePath, nil)
	appropriations, err := client.GetBuildingAppropriations(context.Background(), 121, 3421)
	if err != nil {
		t.Fatalf("GetBuildingAppropriations() error = %v", err)
	}

	want := []models.Apropriacao{
		{Codigo: "A001", Descricao: "Fundacao", Quantidade: 40.5},
		{Codigo: "A002", Descricao: "Estrutura", Quantidade: 15.5},
	}
	if !reflect.DeepEqual(appropriations, want) {
		t.Fatalf("appropriations = %#v, want %#v", appropriations, want)
	}
}

func TestParseAppropriationsSupportsSiengeStockFields(t *testing.T) {
	appropriations, err := parseAppropriations([]byte(`{
		"results": [
			{"buildingUnitId": 3, "sheetItemId": 42, "costEstimationItemReference": "08.004.001", "quantity": 12.3456}
		]
	}`))
	if err != nil {
		t.Fatalf("parseAppropriations() error = %v", err)
	}

	want := []models.Apropriacao{{Codigo: "08.004.001", Descricao: "08.004.001", Referencia: "08.004.001", BuildingUnitID: 3, SheetItemID: 42, Quantidade: 12.3456}}
	if !reflect.DeepEqual(appropriations, want) {
		t.Fatalf("appropriations = %#v, want %#v", appropriations, want)
	}
}

func TestParseAppropriationsDetectsBlockedBudgetItems(t *testing.T) {
	appropriations, err := parseAppropriations([]byte(`{
		"results": [
			{"buildingUnitId": 3, "sheetItemId": 15, "costEstimationItemReference": "08.004.001", "quantity": 12.3456, "isBlocked": true}
		]
	}`))
	if err != nil {
		t.Fatalf("parseAppropriations() error = %v", err)
	}

	if len(appropriations) != 1 || !appropriations[0].Bloqueado {
		t.Fatalf("appropriations = %#v, want blocked appropriation", appropriations)
	}
}

func TestGetBuildingAppropriationsRejectsInvalidIDs(t *testing.T) {
	client := newTestClient(t, "https://example.com"+BasePath, nil)

	if _, err := client.GetBuildingAppropriations(context.Background(), 0, 3421); !errors.Is(err, ErrInvalidCostCenter) {
		t.Fatalf("GetBuildingAppropriations() cost center error = %v, want ErrInvalidCostCenter", err)
	}
	if _, err := client.GetBuildingAppropriations(context.Background(), 121, 0); err == nil {
		t.Fatal("GetBuildingAppropriations() resource error = nil, want error")
	}
}

func TestParseStockItemsSupportsArrayAndAlternativeFieldNames(t *testing.T) {
	items, err := parseStockItems([]byte(`[
		{"supplyId":"3421", "supplyName":"Cimento", "detail":"CP III", "brand":"Votorantim", "balance":"10.5", "unit":"SC"}
	]`))
	if err != nil {
		t.Fatalf("parseStockItems() error = %v", err)
	}

	if len(items) != 1 || items[0].ID != 3421 || items[0].Quantidade != 10.5 || items[0].Marca != "Votorantim" {
		t.Fatalf("items = %#v, want parsed alternative fields", items)
	}
}

func TestParseStockItemsRejectsMissingID(t *testing.T) {
	_, err := parseStockItems([]byte(`{"results":[{"resourceName":"Cimento"}]}`))
	if err == nil {
		t.Fatal("parseStockItems() error = nil, want error")
	}
}
