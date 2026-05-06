package ui

import (
	"errors"
	"net/http"
	"os"
	"testing"

	fynetest "fyne.io/fyne/v2/test"

	"sienge-transfer/api"
	"sienge-transfer/config"
	"sienge-transfer/models"
)

func TestMain(m *testing.M) {
	fynetest.NewApp()
	os.Exit(m.Run())
}

func TestNewAppStateInitializesPersistentTabStates(t *testing.T) {
	cfg := testConfig()
	state := NewAppState(cfg)

	if state.Status != "Pronto." {
		t.Fatalf("Status = %q, want Pronto.", state.Status)
	}
	if state.Obras.UsuarioNome != cfg.Usuario.Nome || state.Obras.UsuarioCargo != cfg.Usuario.Cargo {
		t.Fatalf("Obras state user = %#v, want config user", state.Obras)
	}
	if len(state.Obras.Obras) != 2 {
		t.Fatalf("len(Obras) = %d, want 2", len(state.Obras.Obras))
	}
	if state.Transferencia.CodigoDocumento != "TR" || state.Transferencia.CodigoMovimento != "3" {
		t.Fatalf("transfer defaults = %s/%s, want TR/3", state.Transferencia.CodigoDocumento, state.Transferencia.CodigoMovimento)
	}
}

func TestTabStateIsNotResetWhenContentIsBuiltAgain(t *testing.T) {
	state := NewAppState(testConfig())
	state.Consulta.InsumoIDsInput = "3421"
	state.Consulta.Observacao = "observacao local"
	state.Transferencia.Solicitante = "Maria Santos"

	_ = BuildMainContent(state)
	_ = BuildMainContent(state)

	if state.Consulta.InsumoIDsInput != "3421" || state.Consulta.Observacao != "observacao local" {
		t.Fatalf("consulta state was reset: %#v", state.Consulta)
	}
	if state.Transferencia.Solicitante != "Maria Santos" {
		t.Fatalf("transferencia state was reset: %#v", state.Transferencia)
	}
}

func TestBuildMainContentAndTopBarReturnObjects(t *testing.T) {
	state := NewAppState(testConfig())

	if BuildTopBar(state.Config) == nil {
		t.Fatal("BuildTopBar() returned nil")
	}
	if BuildMainContent(state) == nil {
		t.Fatal("BuildMainContent() returned nil")
	}
}

func TestAsyncRunnerRunsOperationAndDispatchesResult(t *testing.T) {
	dispatched := false
	done := make(chan error, 1)
	runner := NewAsyncRunner(func(fn func()) {
		dispatched = true
		fn()
	})
	wantErr := errors.New("falha")

	runner.Run(func() error { return wantErr }, func(err error) { done <- err })

	if gotErr := <-done; !errors.Is(gotErr, wantErr) {
		t.Fatalf("done error = %v, want %v", gotErr, wantErr)
	}
	if !dispatched {
		t.Fatal("dispatch was not called")
	}
}

func TestStatusMessageForError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{name: "nil", err: nil, want: "Pronto."},
		{name: "missing config", err: config.ErrConfigNotFound, want: "Configuracao inicial nao encontrada"},
		{name: "invalid config", err: config.ErrInvalidConfig, want: "Configuracao local invalida"},
		{name: "api error", err: &api.APIError{StatusCode: http.StatusUnauthorized, Message: "Credenciais invalidas"}, want: "Credenciais invalidas"},
		{name: "unknown", err: errors.New("x"), want: "Ocorreu um erro inesperado"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StatusMessageForError(tt.err)
			if len(got) < len(tt.want) || got[:len(tt.want)] != tt.want {
				t.Fatalf("StatusMessageForError() = %q, want prefix %q", got, tt.want)
			}
		})
	}
}

func testConfig() models.Config {
	return models.Config{
		Usuario: models.Usuario{Nome: "Joao Silva", Cargo: "Engenheiro"},
		Empresa: models.Empresa{Nome: "Construtora XYZ", Subdominio: "construtoraxyz", APIUsuario: "api", APISenha: "senha"},
		Obras: []models.Obra{
			{ID: 121, Nome: "Residencial Novo Horizonte"},
			{ID: 205, Nome: "Comercial Centro"},
		},
	}
}
