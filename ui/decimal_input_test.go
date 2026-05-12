package ui

import "testing"

func TestParseBrazilianDecimal_AcceptsComma(t *testing.T) {
	value, err := ParseBrazilianDecimal("1,5")
	if err != nil || value != 1.5 {
		t.Fatalf("ParseBrazilianDecimal() = %v/%v, want 1.5/nil", value, err)
	}
}

func TestParseBrazilianDecimal_AcceptsDot(t *testing.T) {
	value, err := ParseBrazilianDecimal("1.5")
	if err != nil || value != 1.5 {
		t.Fatalf("ParseBrazilianDecimal() = %v/%v, want 1.5/nil", value, err)
	}
}

func TestFormatBrazilianDecimal_UsesFourDecimals(t *testing.T) {
	if got := FormatBrazilianDecimal(1.5); got != "1,5000" {
		t.Fatalf("FormatBrazilianDecimal() = %q, want 1,5000", got)
	}
}

func TestNormalizeQuantityInput_FormatsOnBlur(t *testing.T) {
	formatted, value, err := NormalizeQuantityInput("1")
	if err != nil || formatted != "1,0000" || value != 1 {
		t.Fatalf("NormalizeQuantityInput() = %q/%v/%v, want 1,0000/1/nil", formatted, value, err)
	}
}

func TestNormalizeQuantityInput_RejectsInvalidText(t *testing.T) {
	if _, _, err := NormalizeQuantityInput("abc"); err == nil {
		t.Fatal("NormalizeQuantityInput() error = nil, want error")
	}
	if _, _, err := NormalizeQuantityInput("0,0000"); err == nil {
		t.Fatal("NormalizeQuantityInput(0) error = nil, want error")
	}
}
