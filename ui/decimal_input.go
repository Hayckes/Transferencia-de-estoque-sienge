package ui

import (
	"errors"
	"strconv"
	"strings"
)

func ParseBrazilianDecimal(input string) (float64, error) {
	input = strings.TrimSpace(input)
	if input == "" || input == "," || input == "." {
		return 0, errors.New("quantidade invalida")
	}
	value, err := strconv.ParseFloat(strings.ReplaceAll(input, ",", "."), 64)
	if err != nil || value <= 0 {
		return 0, errors.New("quantidade invalida")
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
