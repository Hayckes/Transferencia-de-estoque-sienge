package ui

import (
	"errors"
	"strconv"
	"strings"
)

var (
	ErrQuantityRequired       = errors.New("Informe a quantidade a transferir.")
	ErrQuantityInvalidFormat  = errors.New("Quantidade invalida. Informe um valor no formato 0,0000.")
	ErrQuantityMustBePositive = errors.New("A quantidade a transferir deve ser maior que zero.")
)

func ParseBrazilianDecimal(input string) (float64, error) {
	input = strings.TrimSpace(input)
	if input == "" || input == "," || input == "." {
		return 0, ErrQuantityRequired
	}
	value, err := strconv.ParseFloat(strings.ReplaceAll(input, ",", "."), 64)
	if err != nil {
		return 0, ErrQuantityInvalidFormat
	}
	if value <= 0 {
		return 0, ErrQuantityMustBePositive
	}
	return value, nil
}

func FormatBrazilianDecimal(value float64) string {
	return strings.ReplaceAll(strconv.FormatFloat(value, 'f', 4, 64), ".", ",")
}

func NormalizeQuantityInput(input string) (string, float64, error) {
	value, err := ParseBrazilianDecimal(input)
	if err != nil {
		return "", 0, err
	}
	return FormatBrazilianDecimal(value), value, nil
}
