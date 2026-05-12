package models

import "errors"

type TransferStockSnapshotInput struct {
	EstoqueOrigemAntes      float64
	EstoqueDestinoAntes     float64
	ApropriacaoOrigemAntes  *float64
	ApropriacaoDestinoAntes *float64
	Quantidade              float64
}

type TransferStockSnapshot struct {
	EstoqueOrigemAntes       float64
	EstoqueOrigemDepois      float64
	EstoqueDestinoAntes      float64
	EstoqueDestinoDepois     float64
	ApropriacaoOrigemAntes   *float64
	ApropriacaoOrigemDepois  *float64
	ApropriacaoDestinoAntes  *float64
	ApropriacaoDestinoDepois *float64
	QuantidadeEnviada        float64
	QuantidadeRecebida       float64
}

func CalculateTransferStockSnapshot(input TransferStockSnapshotInput) (TransferStockSnapshot, error) {
	if input.Quantidade <= 0 {
		return TransferStockSnapshot{}, errors.New("quantidade deve ser maior que zero")
	}
	if input.Quantidade > input.EstoqueOrigemAntes {
		return TransferStockSnapshot{}, errors.New("quantidade maior que o estoque de origem")
	}
	if input.ApropriacaoOrigemAntes != nil && input.Quantidade > *input.ApropriacaoOrigemAntes {
		return TransferStockSnapshot{}, errors.New("quantidade maior que o saldo da apropriacao de origem")
	}

	snapshot := TransferStockSnapshot{
		EstoqueOrigemAntes:      input.EstoqueOrigemAntes,
		EstoqueOrigemDepois:     input.EstoqueOrigemAntes - input.Quantidade,
		EstoqueDestinoAntes:     input.EstoqueDestinoAntes,
		EstoqueDestinoDepois:    input.EstoqueDestinoAntes + input.Quantidade,
		ApropriacaoOrigemAntes:  cloneFloat(input.ApropriacaoOrigemAntes),
		ApropriacaoDestinoAntes: cloneFloat(input.ApropriacaoDestinoAntes),
		QuantidadeEnviada:       input.Quantidade,
		QuantidadeRecebida:      input.Quantidade,
	}

	if input.ApropriacaoOrigemAntes != nil {
		value := *input.ApropriacaoOrigemAntes - input.Quantidade
		snapshot.ApropriacaoOrigemDepois = &value
	}
	if input.ApropriacaoDestinoAntes != nil {
		value := *input.ApropriacaoDestinoAntes + input.Quantidade
		snapshot.ApropriacaoDestinoDepois = &value
	}

	return snapshot, nil
}

func cloneFloat(value *float64) *float64 {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}
