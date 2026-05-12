package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetStockAppropriationsWithDescriptions_EnrichesAndCaches(t *testing.T) {
	sheetCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/public/api/v1/stock-inventories/121/items/3421/building-appropriation":
			_, _ = w.Write([]byte(`{"results":[
				{"buildingUnitId":3,"sheetItemId":15,"costEstimationItemReference":"00.001","quantity":10},
				{"buildingUnitId":3,"sheetItemId":16,"costEstimationItemReference":"00.002","quantity":5}
			]}`))
		case "/public/api/v1/building-cost-estimations/121/sheets/3/items":
			sheetCalls++
			_, _ = w.Write([]byte(`{"results":[
				{"id":15,"reference":"00.001","description":"Fundacao"},
				{"id":16,"reference":"00.002","description":"Estrutura"}
			]}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client := newTestClient(t, server.URL+BasePath, nil)
	appropriations, err := client.GetStockAppropriationsWithDescriptions(context.Background(), 121, 3421)
	if err != nil {
		t.Fatalf("GetStockAppropriationsWithDescriptions() error = %v", err)
	}
	if sheetCalls != 1 || len(appropriations) != 2 || appropriations[0].Descricao != "Fundacao" || appropriations[1].Descricao != "Estrutura" {
		t.Fatalf("sheetCalls/appropriations = %d/%#v, want enriched with cache", sheetCalls, appropriations)
	}
}
