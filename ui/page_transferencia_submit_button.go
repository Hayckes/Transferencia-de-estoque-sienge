package ui

import "fyne.io/fyne/v2/widget"

type ButtonStyleKind string

const (
	ButtonStyleDefault ButtonStyleKind = "default"
	ButtonStyleSuccess ButtonStyleKind = "success"
)

type ButtonViewModel struct {
	Label string
	Style ButtonStyleKind
}

func BuildTransferSubmitButtonViewModel() ButtonViewModel {
	return ButtonViewModel{Label: "Enviar Transferencia", Style: ButtonStyleSuccess}
}

func NewSuccessButton(label string, tapped func()) *widget.Button {
	button := widget.NewButton(label, tapped)
	button.Importance = widget.SuccessImportance
	return button
}
