package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

const (
	minTypingInputWidth              float32 = 350
	placeholderWidthExtraPads        float32 = 6
	insumoSelectionDialogWidthRatio  float32 = 0.4
	insumoSelectionDialogHeightRatio float32 = 0.6
)

func withMinTypingInputWidth(input fyne.CanvasObject) fyne.CanvasObject {
	return container.NewGridWrap(minTypingInputSize(input), input)
}

func withMinObjectWidth(input fyne.CanvasObject, minWidth float32) fyne.CanvasObject {
	size := input.MinSize()
	if size.Width < minWidth {
		size.Width = minWidth
	}
	return container.NewGridWrap(size, input)
}

func scrollablePage(objects ...fyne.CanvasObject) fyne.CanvasObject {
	content := container.NewPadded(container.NewVBox(objects...))
	scroll := container.NewScroll(content)
	attachVerticalScroll(content, scroll)
	return scroll
}

func flexibleScroll(content fyne.CanvasObject) fyne.CanvasObject {
	scroll := container.NewScroll(content)
	attachVerticalScroll(content, scroll)
	return scroll
}

type horizontalBarScroll struct {
	*container.Scroll
	verticalParent *container.Scroll
}

func horizontalScrollbarOnly(content fyne.CanvasObject) fyne.CanvasObject {
	return &horizontalBarScroll{Scroll: container.NewHScroll(content)}
}

func (s *horizontalBarScroll) Scrolled(ev *fyne.ScrollEvent) {
	if s.verticalParent != nil && ev.Scrolled.DY != 0 {
		s.verticalParent.Scrolled(ev)
	}
}

func attachVerticalScroll(object fyne.CanvasObject, scroll *container.Scroll) {
	switch typed := object.(type) {
	case *horizontalBarScroll:
		typed.verticalParent = scroll
	case *fyne.Container:
		for _, child := range typed.Objects {
			attachVerticalScroll(child, scroll)
		}
	case *container.Scroll:
		attachVerticalScroll(typed.Content, scroll)
	}
}

func responsiveRow(objects ...fyne.CanvasObject) fyne.CanvasObject {
	if len(objects) == 0 {
		return container.NewHBox()
	}
	return container.NewAdaptiveGrid(len(objects), objects...)
}

func expandingInput(input fyne.CanvasObject) fyne.CanvasObject {
	return container.NewBorder(nil, nil, nil, nil, input)
}

func selectableWrappedLabel(text string) *widget.Label {
	label := widget.NewLabel(text)
	label.Wrapping = fyne.TextWrapWord
	label.Selectable = true
	return label
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
