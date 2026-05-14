package ui

import (
	"context"
	"errors"
	"testing"

	"sienge-transfer/models"
)

func TestHandleRecalculateTrigger_ButtonUsesSameRecalculationAsFocusLost(t *testing.T) {
	buttonState := transferStateForRecalculateTrigger("10")
	buttonStock := &fakeStockService{items: []models.Insumo{{ID: 3421, Nome: "Cimento", Quantidade: 20, Unidade: "SC"}}}
	buttonState.Stock = buttonStock
	if err := HandleRecalculateTrigger(context.Background(), buttonState, -1, RecalculateByButton); err != nil {
		t.Fatalf("HandleRecalculateTrigger(button) error = %v", err)
	}

	focusState := transferStateForRecalculateTrigger("10")
	focusStock := &fakeStockService{items: []models.Insumo{{ID: 3421, Nome: "Cimento", Quantidade: 20, Unidade: "SC"}}}
	focusState.Stock = focusStock
	if err := HandleRecalculateTrigger(context.Background(), focusState, 0, RecalculateByQuantityFocusLost); err != nil {
		t.Fatalf("HandleRecalculateTrigger(focus lost) error = %v", err)
	}

	if !buttonStock.itemsCalled || !focusStock.itemsCalled {
		t.Fatalf("button/focus recalculation calls = %v/%v, want both using recalculation service", buttonStock.itemsCalled, focusStock.itemsCalled)
	}
}

func TestHandleRecalculateTrigger_FocusLostRecalculatesWhenQuantityIsValid(t *testing.T) {
	state := transferStateForRecalculateTrigger("1.5")
	stock := &fakeStockService{items: []models.Insumo{{ID: 3421, Nome: "Cimento", Quantidade: 20, Unidade: "SC"}}}
	state.Stock = stock

	if err := HandleRecalculateTrigger(context.Background(), state, 0, RecalculateByQuantityFocusLost); err != nil {
		t.Fatalf("HandleRecalculateTrigger() error = %v", err)
	}
	if state.Transferencia.Itens[0].QuantidadeTransferir != "1,5000" {
		t.Fatalf("QuantidadeTransferir = %q, want normalized 1,5000", state.Transferencia.Itens[0].QuantidadeTransferir)
	}
	if !stock.itemsCalled {
		t.Fatal("stock service should be called after valid focus lost")
	}
}

func TestHandleRecalculateTrigger_FocusLostDoesNotRecalculateWhenQuantityInvalid(t *testing.T) {
	state := transferStateForRecalculateTrigger("abc")
	stock := &fakeStockService{}
	state.Stock = stock

	err := HandleRecalculateTrigger(context.Background(), state, 0, RecalculateByQuantityFocusLost)
	if !errors.Is(err, ErrQuantityInvalidFormat) {
		t.Fatalf("HandleRecalculateTrigger() error = %v, want ErrQuantityInvalidFormat", err)
	}
	if stock.itemsCalled {
		t.Fatal("stock service should not be called when quantity is invalid")
	}
}

func TestHandleRecalculateTrigger_FocusLostRejectsQuantityGreaterThanAvailable(t *testing.T) {
	state := transferStateForRecalculateTrigger("50")
	stock := &fakeStockService{}
	state.Stock = stock

	err := HandleRecalculateTrigger(context.Background(), state, 0, RecalculateByQuantityFocusLost)
	if !errors.Is(err, ErrTransferQuantityGreaterThanAvailable) {
		t.Fatalf("HandleRecalculateTrigger() error = %v, want ErrTransferQuantityGreaterThanAvailable", err)
	}
	if stock.itemsCalled {
		t.Fatal("stock service should not be called when quantity is greater than available")
	}
}

func transferStateForRecalculateTrigger(quantity string) *AppState {
	state := validTransferStateWithItem()
	state.Transferencia.Itens[0].QuantidadeTransferir = quantity
	state.Transferencia.Itens[0].QuantidadeDisponivel = 20
	return state
}
