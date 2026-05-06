package ui

import (
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"sienge-transfer/storage"
)

type HistoricoTabState struct {
	Resumos      []storage.HistoricoResumo
	UltimoStatus string
}

func NewHistoricoTabState() HistoricoTabState {
	return HistoricoTabState{}
}

func BuildHistoricoTab(state *AppState) fyne.CanvasObject {
	status := widget.NewLabel(state.Historico.UltimoStatus)
	refreshButton := widget.NewButton("Atualizar", func() {
		if err := RefreshHistorico(state); err != nil {
			status.SetText(err.Error())
			return
		}
		status.SetText("Historico atualizado.")
	})
	excelButton := widget.NewButton("Abrir Excel", func() {
		if err := OpenHistoricoExcel(state); err != nil {
			status.SetText(err.Error())
			return
		}
		status.SetText("Excel aberto.")
	})

	rows := make([]fyne.CanvasObject, 0, len(state.Historico.Resumos)+1)
	rows = append(rows, widget.NewLabel("Data/Hora | ID Movimento | Solicitante | Origem | Destino | Itens | Total"))
	for _, resumo := range state.Historico.Resumos {
		rows = append(rows, widget.NewLabel(HistoricoResumoRow(resumo)))
	}

	return container.NewVBox(
		widget.NewLabel("Historico resumido"),
		container.NewHBox(refreshButton, excelButton),
		status,
		container.NewVBox(rows...),
	)
}

func RefreshHistorico(state *AppState) error {
	if state.HistoryStore == nil {
		return errors.New("armazenamento de historico nao configurado")
	}

	resumos, err := state.HistoryStore.ReadHistorySummary()
	if err != nil {
		return err
	}

	state.Historico.Resumos = append([]storage.HistoricoResumo(nil), resumos...)
	state.Historico.UltimoStatus = "Historico atualizado."
	return nil
}

func OpenHistoricoExcel(state *AppState) error {
	if state.HistoryStore == nil {
		return errors.New("armazenamento de historico nao configurado")
	}
	if state.FileOpener == nil {
		return errors.New("abridor de arquivos nao configurado")
	}
	if err := state.HistoryStore.EnsureExcelFromHistory(); err != nil {
		return err
	}

	return state.FileOpener.Open(state.HistoryStore.ExcelPath())
}

func HistoricoResumoRow(resumo storage.HistoricoResumo) string {
	return fmt.Sprintf(
		"%s | %s | %s | %s | %s | %d | %s",
		resumo.DataHora,
		resumo.IDMovimento,
		resumo.Solicitante,
		resumo.ObraOrigem,
		resumo.ObraDestino,
		resumo.QuantidadeItens,
		storageTotalQuantity(resumo.TotalQuantidade),
	)
}

func storageTotalQuantity(value float64) string {
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.6f", value), "0"), ".")
}

type SystemFileOpener struct{}

func (SystemFileOpener) Open(path string) error {
	var command *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		command = exec.Command("rundll32", "url.dll,FileProtocolHandler", path)
	case "darwin":
		command = exec.Command("open", path)
	default:
		command = exec.Command("xdg-open", path)
	}

	return command.Start()
}
