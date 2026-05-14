package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"sienge-transfer/models"
)

func TestGetCostCentersCallsEndpointAndParsesSingleObject(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		if r.URL.String() != "/public/api/v1/cost-centers/121" {
			t.Fatalf("URL = %s, want cost center endpoint", r.URL.String())
		}

		_, _ = w.Write([]byte(`{"id":121,"description":"Residencial Novo Horizonte"}`))
	}))
	defer server.Close()

	client := newTestClient(t, server.URL+BasePath, nil)
	centers, err := client.GetCostCenters(context.Background(), 121)
	if err != nil {
		t.Fatalf("GetCostCenters() error = %v", err)
	}

	want := []models.Obra{{ID: 121, Nome: "Residencial Novo Horizonte"}}
	if !reflect.DeepEqual(centers, want) {
		t.Fatalf("centers = %#v, want %#v", centers, want)
	}
}

func TestParseCostCentersSupportsMultipleResults(t *testing.T) {
	centers, err := parseCostCenters([]byte(`{
		"results": [
			{"costCenterId": 121, "name": "Residencial"},
			{"costCenterCode": "205", "costCenterDescription": "Comercial Centro"}
		]
	}`), 0)
	if err != nil {
		t.Fatalf("parseCostCenters() error = %v", err)
	}

	want := []models.Obra{{ID: 121, Nome: "Residencial"}, {ID: 205, Nome: "Comercial Centro"}}
	if !reflect.DeepEqual(centers, want) {
		t.Fatalf("centers = %#v, want %#v", centers, want)
	}
}

func TestGetCostCentersMapsNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message":"not found"}`))
	}))
	defer server.Close()

	client := newTestClient(t, server.URL+BasePath, nil)
	_, err := client.GetCostCenters(context.Background(), 999)
	if !errors.Is(err, ErrCostCenterNotFound) {
		t.Fatalf("GetCostCenters() error = %v, want ErrCostCenterNotFound", err)
	}
}

func TestGetCostCentersRejectsInvalidID(t *testing.T) {
	client := newTestClient(t, "https://example.com"+BasePath, nil)

	_, err := client.GetCostCenters(context.Background(), 0)
	if !errors.Is(err, ErrInvalidCostCenter) {
		t.Fatalf("GetCostCenters() error = %v, want ErrInvalidCostCenter", err)
	}
}
