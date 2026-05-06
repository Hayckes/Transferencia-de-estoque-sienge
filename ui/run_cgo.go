//go:build cgo

package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/widget"

	"sienge-transfer/config"
)

func Run() {
	fyneApp := app.NewWithID(appID)
	window := fyneApp.NewWindow("Transferencia de Estoque Sienge")
	window.Resize(fyne.NewSize(1280, 720))

	store, err := config.DefaultStore()
	if err != nil {
		window.SetContent(widget.NewLabel(StatusMessageForError(err)))
		window.ShowAndRun()
		return
	}

	if needsOnboarding, err := NeedsOnboarding(store); err != nil {
		window.SetContent(BuildFatalErrorContent(StatusMessageForError(err)))
		window.ShowAndRun()
		return
	} else if needsOnboarding {
		window.SetContent(BuildOnboardingContent(window, store, func(cfg configLoaded) {
			state := NewConfiguredAppState(cfg.Config, store, window)
			window.SetContent(BuildMainContent(state))
		}))
		window.ShowAndRun()
		return
	}

	cfg, err := store.Load()
	if err != nil {
		window.SetContent(BuildFatalErrorContent(StatusMessageForError(err)))
		window.ShowAndRun()
		return
	}

	state := NewConfiguredAppState(cfg, store, window)
	window.SetContent(BuildMainContent(state))
	window.ShowAndRun()
}
