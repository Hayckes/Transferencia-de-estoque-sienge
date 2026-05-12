package models

import "math"

const appropriationReconciliationTolerance = 0.0001

type ReconciliationResult struct {
	OK                     bool
	StockQuantity          float64
	AppropriationsQuantity float64
	Difference             float64
}

func ReconcileStockAndAppropriations(stockQty float64, appropriations []Apropriacao) ReconciliationResult {
	sum := 0.0
	for _, appropriation := range appropriations {
		sum += appropriation.Quantidade
	}
	difference := sum - stockQty
	return ReconciliationResult{
		OK:                     math.Abs(difference) <= appropriationReconciliationTolerance,
		StockQuantity:          stockQty,
		AppropriationsQuantity: sum,
		Difference:             difference,
	}
}
