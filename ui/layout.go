package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

const (
	minTypingInputWidth              float32 = 180
	placeholderWidthExtraPads        float32 = 6
	insumoSelectionDialogWidthRatio  float32 = 0.4
	insumoSelectionDialogHeightRatio float32 = 0.6
)

func withMinTypingInputWidth(input fyne.CanvasObject) fyne.CanvasObject {
	return container.NewGridWrap(minTypingInputSize(input), input)
}

func minTypingInputSize(input fyne.CanvasObject) fyne.Size {
	size := input.MinSize()
	if size.Width < minTypingInputWidth {
		size.Width = minTypingInputWidth
	}
	if entry, ok := input.(*widget.Entry); ok && entry.PlaceHolder != "" {
		placeholderWidth := widget.NewLabel(entry.PlaceHolder).MinSize().Width + theme.Padding()*placeholderWidthExtraPads
		if size.Width < placeholderWidth {
			size.Width = placeholderWidth
		}
	}

	return size
}

func sizeAtLeastWindowRatio(contentMin, windowSize fyne.Size, widthRatio, heightRatio float32) fyne.Size {
	size := contentMin
	if windowSize.Width > 0 {
		minWidth := windowSize.Width * widthRatio
		if size.Width < minWidth {
			size.Width = minWidth
		}
	}
	if windowSize.Height > 0 {
		minHeight := windowSize.Height * heightRatio
		if size.Height < minHeight {
			size.Height = minHeight
		}
	}

	return size
}
