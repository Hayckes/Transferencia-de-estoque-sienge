//go:build cgo

package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/widget"

	"sienge-transfer/api"
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

	cfg, err := store.Load()
	if err != nil {
		window.SetContent(widget.NewLabel("Configuracao inicial necessaria. A interface de onboarding sera aberta na proxima etapa."))
		window.ShowAndRun()
		return
	}

	state := NewAppStateWithStore(cfg, store)
	client, err := api.NewClient(cfg.Empresa.Subdominio, cfg.Empresa.APIUsuario, cfg.Empresa.APISenha)
	if err == nil {
		state.Stock = client
	}
	state.Runner = NewAsyncRunner(fyne.Do)
	window.SetContent(BuildMainContent(state))
	window.ShowAndRun()
}
