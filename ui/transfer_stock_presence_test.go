package ui

import (
	"strings"
	"testing"
)

func TestBuildStockPresenceFeedback_FoundInBoth(t *testing.T) {
	if !strings.Contains(BuildStockPresenceFeedback(true, true), "origem e no destino") {
		t.Fatalf("feedback mismatch")
	}
}

func TestBuildStockPresenceFeedback_MissingInOrigin(t *testing.T) {
	if !strings.Contains(BuildStockPresenceFeedback(false, true), "sem saldo de origem") {
		t.Fatalf("feedback mismatch")
	}
}

func TestBuildStockPresenceFeedback_MissingInDestination(t *testing.T) {
	if !strings.Contains(BuildStockPresenceFeedback(true, false), "nao existe no estoque da obra de destino") {
		t.Fatalf("feedback mismatch")
	}
}

func TestBuildStockPresenceFeedback_MissingInBoth(t *testing.T) {
	if !strings.Contains(BuildStockPresenceFeedback(false, false), "origem nem na obra de destino") {
		t.Fatalf("feedback mismatch")
	}
}
