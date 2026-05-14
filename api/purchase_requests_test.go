package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetPurchaseRequestItems_BuildsURLParsesAndDeduplicates(t *testing.T) {
	called := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called++
		if r.URL.String() != "/public/api/v1/purchase-requests/all/items?purchaseRequestId=2291&buildingId=125&limit=100&offset=0" {
			t.Fatalf("URL = %s", r.URL.String())
		}
		_, _ = w.Write([]byte(`{"results":[
			{"resourceId":1001,"resourceName":"Cimento","detailId":5,"trademarkId":8,"quantity":3},
			{"resourceId":1001,"resourceName":"Cimento","detailId":5,"trademarkId":8,"quantity":3}
		]}`))
	}))
	defer server.Close()

	client := newTestClient(t, server.URL+BasePath, nil)
	items, err := client.GetPurchaseRequestItems(context.Background(), 2291, 125)
	if err != nil {
		t.Fatalf("GetPurchaseRequestItems() error = %v", err)
	}
	if called != 1 || len(items) != 1 || items[0].ResourceID != 1001 || items[0].DetailID != 5 || items[0].BrandID != 8 {
		t.Fatalf("called/items = %d/%#v, want one deduplicated item", called, items)
	}
}

func TestParsePurchaseRequestItems_ParsesArrayResponse(t *testing.T) {
	items, err := parsePurchaseRequestItems([]byte(`[{"supplyId":"1001","supplyName":"Cimento"}]`), 1, 2)
	if err != nil {
		t.Fatalf("parsePurchaseRequestItems() error = %v", err)
	}
	if len(items) != 1 || items[0].ResourceID != 1001 || items[0].PurchaseRequestID != 1 || items[0].BuildingID != 2 {
		t.Fatalf("items = %#v, want parsed array response", items)
	}
}
