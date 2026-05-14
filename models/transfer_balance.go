package models

import "errors"

type TransferBalanceInput struct {
	OriginTotalStock              float64
	DestinationTotalStock         float64
	OriginAppropriationStock      *float64
	DestinationAppropriationStock *float64
	QuantityToTransfer            float64
}

type TransferBalanceOutput struct {
	OriginCurrentStock           float64
	DestinationCurrentStock      float64
	OriginAfterTransfer          float64
	DestinationAfterTransfer     float64
	UsesOriginAppropriation      bool
	UsesDestinationAppropriation bool
}

func CalculateTransferBalances(input TransferBalanceInput) (TransferBalanceOutput, error) {
	if input.QuantityToTransfer <= 0 {
		return TransferBalanceOutput{}, errors.New("quantidade deve ser maior que zero")
	}
	originCurrent := input.OriginTotalStock
	usesOriginAppropriation := input.OriginAppropriationStock != nil
	if usesOriginAppropriation {
		originCurrent = *input.OriginAppropriationStock
	}
	destinationCurrent := input.DestinationTotalStock
	usesDestinationAppropriation := input.DestinationAppropriationStock != nil
	if usesDestinationAppropriation {
		destinationCurrent = *input.DestinationAppropriationStock
	}
	originAfter := originCurrent - input.QuantityToTransfer
	if originAfter < 0 {
		return TransferBalanceOutput{}, errors.New("saldo de origem ficaria negativo")
	}
	return TransferBalanceOutput{
		OriginCurrentStock:           originCurrent,
		DestinationCurrentStock:      destinationCurrent,
		OriginAfterTransfer:          originAfter,
		DestinationAfterTransfer:     destinationCurrent + input.QuantityToTransfer,
		UsesOriginAppropriation:      usesOriginAppropriation,
		UsesDestinationAppropriation: usesDestinationAppropriation,
	}, nil
}
