package ui

import (
	"errors"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"

	"sienge-transfer/models"
)

type InsumoTableRow struct {
	ObraLabel   string
	ID          string
	Nome        string
	Detalhe     string
	Marca       string
	Unidade     string
	Quantidade  string
	ActionLabel string
	OnAction    func()
}

func ConsultaInsumoTableColumns() []string {
	return []string{"Obra", "ID", "Nome", "Detalhe", "Marca", "Unidade", "Qtd. em Estoque", "Acoes"}
}

func TransferInsumoSelectionTableColumns() []string {
	return []string{"ID", "Nome", "Detalhe", "Marca", "Unidade", "Qtd. em Estoque"}
}

func BuildConsultaInsumoRows(results []models.ConsultaResultado) []InsumoTableRow {
	rows := make([]InsumoTableRow, 0, len(results))
	for _, result := range results {
		rows = append(rows, InsumoTableRow{
			ObraLabel:   models.Obra{ID: result.ObraID, Nome: result.ObraNome}.Label(),
			ID:          strconv.Itoa(result.InsumoID),
			Nome:        result.InsumoNome,
			Detalhe:     result.Detalhe,
			Marca:       result.Marca,
			Unidade:     result.Unidade,
			Quantidade:  models.FormatQuantidade(result.Quantidade, result.Unidade),
			ActionLabel: "Detalhes",
		})
	}
	return rows
}

func BuildTransferInsumoSelectionRows(items []models.Insumo) []InsumoTableRow {
	rows := make([]InsumoTableRow, 0, len(items))
	for _, item := range items {
		rows = append(rows, InsumoTableRow{
			ID:         strconv.Itoa(item.ID),
			Nome:       item.Nome,
			Detalhe:    item.Detalhe,
			Marca:      item.Marca,
			Unidade:    item.Unidade,
			Quantidade: models.FormatQuantidade(item.Quantidade, item.Unidade),
		})
	}
	return rows
}

func ResolveSelectedInsumo(items []models.Insumo, selectedIndex int) (models.Insumo, error) {
	if selectedIndex < 0 || selectedIndex >= len(items) {
		return models.Insumo{}, errors.New("selecione um insumo valido")
	}
	return items[selectedIndex], nil
}

func NewInsumoResultsTable(rows []InsumoTableRow, onSelected ...func(int)) fyne.CanvasObject {
	rowsCopy := append([]InsumoTableRow(nil), rows...)
	return newInsumoResultsTable(&rowsCopy, firstSelectHandler(onSelected))
}

func newInsumoResultsTable(rows *[]InsumoTableRow, onSelected func(int)) *widget.Table {
	columns := ConsultaInsumoTableColumns()
	table := newInsumoTable(columns, rows, func(row InsumoTableRow, col int) string {
		switch col {
		case 0:
			return row.ObraLabel
		case 1:
			return row.ID
		case 2:
			return row.Nome
		case 3:
			return row.Detalhe
		case 4:
			return row.Marca
		case 5:
			return row.Unidade
		case 6:
			return row.Quantidade
		case 7:
			return row.ActionLabel
		default:
			return ""
		}
	}, onSelected)

	for col, width := range []float32{240, 80, 220, 160, 160, 80, 150, 110} {
		table.SetColumnWidth(col, width)
	}
	return table
}

func NewInsumoSelectionTable(rows []InsumoTableRow, onSelected ...func(int)) fyne.CanvasObject {
	rowsCopy := append([]InsumoTableRow(nil), rows...)
	return newInsumoSelectionTable(&rowsCopy, firstSelectHandler(onSelected))
}

func newInsumoSelectionTable(rows *[]InsumoTableRow, onSelected func(int)) *widget.Table {
	columns := TransferInsumoSelectionTableColumns()
	table := newInsumoTable(columns, rows, func(row InsumoTableRow, col int) string {
		switch col {
		case 0:
			return row.ID
		case 1:
			return row.Nome
		case 2:
			return row.Detalhe
		case 3:
			return row.Marca
		case 4:
			return row.Unidade
		case 5:
			return row.Quantidade
		default:
			return ""
		}
	}, onSelected)

	for col, width := range []float32{80, 220, 160, 160, 80, 150} {
		table.SetColumnWidth(col, width)
	}
	return table
}

func newInsumoTable(columns []string, rows *[]InsumoTableRow, value func(InsumoTableRow, int) string, onSelected func(int)) *widget.Table {
	table := widget.NewTable(
		func() (int, int) { return len(*rows) + 1, len(columns) },
		func() fyne.CanvasObject {
			label := widget.NewLabel("")
			label.Wrapping = fyne.TextTruncate
			return label
		},
		func(id widget.TableCellID, obj fyne.CanvasObject) {
			label := obj.(*widget.Label)
			label.TextStyle = fyne.TextStyle{}
			if id.Row == 0 {
				label.TextStyle = fyne.TextStyle{Bold: true}
				label.SetText(columns[id.Col])
				return
			}
			label.SetText(value((*rows)[id.Row-1], id.Col))
		},
	)
	table.OnSelected = func(id widget.TableCellID) {
		if id.Row == 0 || onSelected == nil {
			return
		}
		onSelected(id.Row - 1)
	}
	return table
}

func firstSelectHandler(handlers []func(int)) func(int) {
	if len(handlers) == 0 {
		return nil
	}
	return handlers[0]
}
