package ui

import (
	"context"

	"fyne.io/fyne/v2"

	"sienge-transfer/api"
	"sienge-transfer/config"
	"sienge-transfer/models"
	"sienge-transfer/storage"
)

type configLoaded struct {
	Config models.Config
}

type SiengeCredentialValidator struct{}

func (SiengeCredentialValidator) ValidateCredentials(ctx context.Context, empresa models.Empresa) error {
	client, err := api.NewClient(empresa.Subdominio, empresa.APIUsuario, empresa.APISenha)
	if err != nil {
		return err
	}

	return client.ValidateCredentials(ctx)
}

func NewConfiguredAppState(cfg models.Config, store config.Store, window fyne.Window) *AppState {
	state := NewAppStateWithStore(cfg, store)
	state.Window = window
	state.Runner = NewAsyncRunner(fyne.Do)
	state.FileOpener = SystemFileOpener{}
	dataStore := storage.NewStore(store.Dir)
	state.TransferStore = dataStore
	state.HistoryStore = dataStore
	ConfigureAPIClient(state)
	state.RefreshUI = func() {
		window.SetContent(BuildMainContent(state))
	}
	_ = RefreshHistorico(state)
	return state
}

func ConfigureAPIClient(state *AppState) error {
	client, err := api.NewClient(state.Config.Empresa.Subdominio, state.Config.Empresa.APIUsuario, state.Config.Empresa.APISenha)
	if err != nil {
		state.Stock = nil
		state.CostCenters = nil
		state.PurchaseRequests = nil
		state.Transfer = nil
		return err
	}

	state.Stock = client
	state.CostCenters = client
	state.PurchaseRequests = client
	state.Transfer = client
	return nil
}
