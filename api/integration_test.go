//go:build integration

package api

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestSiengeCredentialsIntegration(t *testing.T) {
	subdomain := os.Getenv("SIENGE_SUBDOMAIN")
	username := os.Getenv("SIENGE_USER")
	password := os.Getenv("SIENGE_PASSWORD")
	if subdomain == "" || username == "" || password == "" {
		t.Skip("defina SIENGE_SUBDOMAIN, SIENGE_USER e SIENGE_PASSWORD para rodar este teste")
	}

	client, err := NewClient(subdomain, username, password)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := client.ValidateCredentials(ctx); err != nil {
		t.Fatalf("ValidateCredentials() error = %v", err)
	}
}
