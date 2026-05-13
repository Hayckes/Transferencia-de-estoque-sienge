package ui

import "fyne.io/fyne/v2/widget"

type QuantityEntry struct {
	widget.Entry
	OnFocusLost func(value string)
}

func NewQuantityEntry(onFocusLost func(value string)) *QuantityEntry {
	entry := &QuantityEntry{OnFocusLost: onFocusLost}
	entry.ExtendBaseWidget(entry)
	return entry
}

func (entry *QuantityEntry) FocusLost() {
	entry.Entry.FocusLost()
	if entry.OnFocusLost != nil {
		entry.OnFocusLost(entry.Text)
	}
}
