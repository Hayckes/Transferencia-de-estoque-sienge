package ui

import (
	"errors"
	"net/http"
	"os"
	"testing"

	"fyne.io/fyne/v2"
	fynetest "fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

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
	if state.ActiveTab != TabObras {
		t.Fatalf("ActiveTab = %q, want %q", state.ActiveTab, TabObras)
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
	state.Transferencia.Solicitante = "Maria Santos"

	_ = BuildMainContent(state)
	_ = BuildMainContent(state)

	if state.Consulta.InsumoIDsInput != "3421" {
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

func TestBuildMainTabsPreservesSelectedTab(t *testing.T) {
	state := NewAppState(testConfig())
	tabs := BuildMainTabs(state)
	tabs.SelectIndex(1)
	if state.ActiveTab != TabConsulta {
		t.Fatalf("ActiveTab after select = %q, want %q", state.ActiveTab, TabConsulta)
	}

	rebuilt := BuildMainTabs(state)
	if rebuilt.Selected() == nil || rebuilt.Selected().Text != TabConsulta {
		t.Fatalf("rebuilt selected tab = %#v, want Consulta", rebuilt.Selected())
	}
}

const compactWindowMaxMinWidth float32 = 520

func TestBuildMainTabsAllowsCompactWindowWidth(t *testing.T) {
	state := NewAppState(testConfig())
	state.Transferencia.Itens = validTransferStateWithItem().Transferencia.Itens

	minSize := BuildMainTabs(state).MinSize()
	if minSize.Width > compactWindowMaxMinWidth {
		t.Fatalf("BuildMainTabs().MinSize().Width = %v, want at most %v", minSize.Width, compactWindowMaxMinWidth)
	}
}

func TestWithMinTypingInputWidthEnforcesMinimumWidth(t *testing.T) {
	entry := widget.NewEntry()
	wrapped := withMinTypingInputWidth(entry)

	if wrapped.MinSize().Width < minTypingInputWidth {
		t.Fatalf("wrapped input width = %v, want at least %v", wrapped.MinSize().Width, minTypingInputWidth)
	}
}

func TestWithMinTypingInputWidthUsesPlaceholderWidth(t *testing.T) {
	entry := widget.NewEntry()
	entry.SetPlaceHolder("IDs dos insumos separados por virgula ou espaco")
	wrapped := withMinTypingInputWidth(entry)
	wantMinWidth := widget.NewLabel(entry.PlaceHolder).MinSize().Width + theme.Padding()*placeholderWidthExtraPads

	if wrapped.MinSize().Width < wantMinWidth {
		t.Fatalf("wrapped input width = %v, want at least placeholder width %v", wrapped.MinSize().Width, wantMinWidth)
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

func TestAppStateRefreshCoalescesQueuedRefreshes(t *testing.T) {
	state := NewAppState(testConfig())
	queued := make([]func(), 0)
	state.Runner = NewAsyncRunner(func(fn func()) { queued = append(queued, fn) })
	rootRefreshes := 0
	tabRefreshes := make([]string, 0)
	state.RefreshUI = func() { rootRefreshes++ }
	state.RefreshTabUI = func(tab string) { tabRefreshes = append(tabRefreshes, tab) }

	state.RefreshTab(TabConsulta)
	state.RefreshTab(TabEmprestimos)

	if len(queued) != 1 {
		t.Fatalf("queued refreshes = %d, want 1", len(queued))
	}
	if rootRefreshes != 0 || len(tabRefreshes) != 0 {
		t.Fatalf("refreshes before dispatch = %d/%d, want 0/0", rootRefreshes, len(tabRefreshes))
	}
	queued[0]()
	if rootRefreshes != 0 || len(tabRefreshes) != 1 || tabRefreshes[0] != TabEmprestimos {
		t.Fatalf("refreshes after dispatch = root %d tabs %#v, want one Emprestimos tab refresh", rootRefreshes, tabRefreshes)
	}
	if state.ActiveTab != TabEmprestimos {
		t.Fatalf("ActiveTab = %q, want %q", state.ActiveTab, TabEmprestimos)
	}
}

func TestAppStateFullRefreshSupersedesQueuedTabRefresh(t *testing.T) {
	state := NewAppState(testConfig())
	queued := make([]func(), 0)
	state.Runner = NewAsyncRunner(func(fn func()) { queued = append(queued, fn) })
	rootRefreshes := 0
	tabRefreshes := 0
	state.RefreshUI = func() { rootRefreshes++ }
	state.RefreshTabUI = func(string) { tabRefreshes++ }

	state.Refresh()
	state.RefreshTab(TabEmprestimos)

	if len(queued) != 1 {
		t.Fatalf("queued refreshes = %d, want 1", len(queued))
	}
	queued[0]()
	if rootRefreshes != 1 || tabRefreshes != 0 {
		t.Fatalf("refreshes after dispatch = root %d tabs %d, want root 1 tabs 0", rootRefreshes, tabRefreshes)
	}
	if state.ActiveTab != TabEmprestimos {
		t.Fatalf("ActiveTab = %q, want %q", state.ActiveTab, TabEmprestimos)
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

func TestStatusViewUsesSelectableWrappedMessageAndActions(t *testing.T) {
	status := NewStatusView(nil, "")
	if !status.label.Selectable || status.label.Wrapping != fyne.TextWrapWord {
		t.Fatalf("status label selectable/wrapping = %v/%v, want selectable word wrap", status.label.Selectable, status.label.Wrapping)
	}
	if !status.copyButton.Disabled() || !status.detailsButton.Disabled() {
		t.Fatal("empty status should disable copy/details actions")
	}

	status.SetText("Erro detalhado para copiar")
	if status.copyButton.Disabled() {
		t.Fatal("non-empty status should enable copy action")
	}
	if !status.detailsButton.Disabled() {
		t.Fatal("details action should stay disabled without a window")
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
