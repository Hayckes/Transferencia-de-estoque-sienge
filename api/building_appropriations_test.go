package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"sienge-transfer/models"
)

func TestGetBuildingAppropriations_IncludesDetailAndTrademarkFilters(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/public/api/v1/stock-inventories/111/items/1001/building-appropriation" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		query := r.URL.Query()
		if query.Get("detailId") != "123" || query.Get("trademarkId") != "456" || query.Get("offset") != "0" || query.Get("limit") != "100" {
			t.Fatalf("query = %s, want detail/trademark/offset/limit", r.URL.RawQuery)
		}
		_, _ = w.Write([]byte(`{"results":[]}`))
	}))
	defer server.Close()

	client := newTestClient(t, server.URL+BasePath, nil)
	detailID := 123
	trademarkID := 456
	if _, err := client.GetBuildingAppropriationsByQuery(context.Background(), BuildingAppropriationQuery{CostCenterID: 111, ResourceID: 1001, DetailID: &detailID, TrademarkID: &trademarkID}); err != nil {
		t.Fatalf("GetBuildingAppropriationsByQuery() error = %v", err)
	}
}

func TestGetBuildingAppropriations_OmitsEmptyOptionalFilters(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if query.Has("detailId") || query.Has("trademarkId") {
			t.Fatalf("query = %s, want no optional filters", r.URL.RawQuery)
		}
		_, _ = w.Write([]byte(`{"results":[]}`))
	}))
	defer server.Close()

	client := newTestClient(t, server.URL+BasePath, nil)
	if _, err := client.GetBuildingAppropriations(context.Background(), 111, 1001); err != nil {
		t.Fatalf("GetBuildingAppropriations() error = %v", err)
	}
}

func TestGetBuildingAppropriations_UsesTrademarkIdNotBrandName(t *testing.T) {
	path := buildBuildingAppropriationPath(BuildingAppropriationQuery{CostCenterID: 111, ResourceID: 1001, TrademarkID: intPtr(456)})
	if path == "" || !strings.Contains(path, "trademarkId=456") || strings.Contains(path, "Votoran") {
		t.Fatalf("path = %q, want numeric trademarkId only", path)
	}
}

func TestAppropriationCacheKey_IncludesDetailAndTrademark(t *testing.T) {
	first := StockItemCacheKey(111, models.Insumo{ID: 1001, DetalheID: 123, MarcaID: 456})
	second := StockItemCacheKey(111, models.Insumo{ID: 1001, DetalheID: 123})
	if first == second {
		t.Fatalf("keys should differ: %#v %#v", first, second)
	}
}

func intPtr(value int) *int { return &value }
