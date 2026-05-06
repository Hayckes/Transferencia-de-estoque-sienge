package models

import (
	"errors"
	"strconv"
	"strings"
)

var ErrIDsInsumoObrigatorios = errors.New("informe pelo menos um ID de insumo")

func ParseInsumoIDs(input string) ([]int, error) {
	parts := strings.FieldsFunc(input, func(r rune) bool {
		return r == ',' || r == ' ' || r == '\t' || r == '\n' || r == '\r'
	})

	ids := make([]int, 0, len(parts))
	seen := make(map[int]bool, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		id, err := strconv.Atoi(part)
		if err != nil || id <= 0 {
			return nil, errors.New("IDs de insumo devem conter apenas numeros positivos")
		}

		if !seen[id] {
			ids = append(ids, id)
			seen[id] = true
		}
	}

	if len(ids) == 0 {
		return nil, ErrIDsInsumoObrigatorios
	}

	return ids, nil
}

func FormatQuantidade(value float64, unidade string) string {
	formatted := strconv.FormatFloat(value, 'f', -1, 64)
	if strings.TrimSpace(unidade) == "" {
		return formatted
	}

	return formatted + " " + strings.TrimSpace(unidade)
}
