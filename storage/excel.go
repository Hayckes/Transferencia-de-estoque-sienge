package storage

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"

	"sienge-transfer/models"
)

const (
	TransferenciasExcelFileName = "transferencias.xlsx"
	excelSheetName              = "Transferencias"
)

var ExcelHeaders = []string{
	"ID de Transferencia",
	"Data/Hora",
	"Usuario",
	"Cargo",
	"Solicitante",
	"Observacao",
	"ID Obra Origem",
	"Nome Obra Origem",
	"Apropriacao Origem",
	"Quantidade Origem no Momento da Transferencia",
	"Quantidade Enviada",
	"Quantidade Origem Apos Transferencia",
	"Quantidade Apropriacao Origem no Momento da Transferencia",
	"Quantidade Apropriacao Origem Apos Transferencia",
	"ID Obra Destino",
	"Nome Obra Destino",
	"Apropriacao Destino Codigo",
	"Apropriacao Destino Descricao",
	"Quantidade Destino no Momento da Transferencia",
	"Quantidade Recebida",
	"Quantidade Destino Apos Transferencia",
	"Quantidade Apropriacao Destino no Momento da Transferencia",
	"Quantidade Apropriacao Destino Apos Transferencia",
	"Insumo ID",
	"Nome do Insumo",
	"Detalhe",
	"Marca",
	"Unidade",
	"Tipo da Transferencia",
	"Status do Emprestimo",
}

func (s Store) EnsureExcelFromHistory() error {
	_, err := os.Stat(s.ExcelPath())
	if err == nil {
		matches, err := excelHeadersMatch(s.ExcelPath())
		if err != nil {
			return err
		}
		if matches {
			styled, err := excelLoanStatusStylesMatch(s.ExcelPath())
			if err != nil {
				return err
			}
			if styled {
				return nil
			}
		}
		history, err := s.ReadHistory()
		if err != nil {
			return err
		}
		return s.RebuildExcel(history)
	}
	if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	history, err := s.ReadHistory()
	if err != nil {
		return err
	}

	return s.RebuildExcel(history)
}

func excelHeadersMatch(path string) (bool, error) {
	file, err := excelize.OpenFile(path)
	if err != nil {
		return false, err
	}
	defer file.Close()

	rows, err := file.GetRows(excelSheetName)
	if err != nil {
		return false, err
	}
	if len(rows) == 0 || len(rows[0]) != len(ExcelHeaders) {
		return false, nil
	}
	for index, header := range ExcelHeaders {
		if rows[0][index] != header {
			return false, nil
		}
	}

	return true, nil
}

func excelLoanStatusStylesMatch(path string) (bool, error) {
	file, err := excelize.OpenFile(path)
	if err != nil {
		return false, err
	}
	defer file.Close()

	rows, err := file.GetRows(excelSheetName)
	if err != nil {
		return false, err
	}
	for index, row := range rows {
		if index == 0 || len(row) < len(ExcelHeaders) {
			continue
		}
		wantColor, ok := loanStatusExcelFillColor(row[len(ExcelHeaders)-1])
		if !ok {
			continue
		}
		cell, err := excelize.CoordinatesToCellName(len(ExcelHeaders), index+1)
		if err != nil {
			return false, err
		}
		styleID, err := file.GetCellStyle(excelSheetName, cell)
		if err != nil {
			return false, err
		}
		style, err := file.GetStyle(styleID)
		if err != nil {
			return false, err
		}
		if !excelStyleHasFillColor(style, wantColor) {
			return false, nil
		}
	}
	return true, nil
}

func (s Store) RebuildExcel(history []models.Transferencia) error {
	if err := s.EnsureDir(); err != nil {
		return err
	}

	file := excelize.NewFile()
	defer file.Close()

	defaultSheet := file.GetSheetName(0)
	if err := file.SetSheetName(defaultSheet, excelSheetName); err != nil {
		return err
	}
	if err := writeExcelHeaders(file); err != nil {
		return err
	}

	nextRow := 2
	transferID := 1
	for _, transfer := range history {
		for _, item := range transfer.Insumos {
			if err := writeExcelRow(file, nextRow, transferID, transfer, item); err != nil {
				return err
			}
			nextRow++
		}
		transferID++
	}

	return saveExcelAtomically(file, s.ExcelPath())
}

func (s Store) AppendTransferToExcel(transfer models.Transferencia) error {
	if err := s.EnsureExcelFromHistory(); err != nil {
		return err
	}

	file, err := excelize.OpenFile(s.ExcelPath())
	if err != nil {
		return err
	}
	defer file.Close()

	sheetIndex, err := file.GetSheetIndex(excelSheetName)
	if err != nil {
		return err
	}
	if sheetIndex == -1 {
		index, err := file.NewSheet(excelSheetName)
		if err != nil {
			return err
		}
		file.SetActiveSheet(index)
		if err := writeExcelHeaders(file); err != nil {
			return err
		}
	}

	nextRow, err := nextExcelRow(file)
	if err != nil {
		return err
	}
	transferID, err := nextExcelTransferID(file)
	if err != nil {
		return err
	}
	for _, item := range transfer.Insumos {
		if err := writeExcelRow(file, nextRow, transferID, transfer, item); err != nil {
			return err
		}
		nextRow++
	}

	return saveExcelAtomically(file, s.ExcelPath())
}

func saveExcelAtomically(file *excelize.File, path string) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp-*.xlsx")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	defer os.Remove(tmpName)

	if err := file.SaveAs(tmpName); err != nil {
		return err
	}
	return replaceFile(tmpName, path)
}

func writeExcelHeaders(file *excelize.File) error {
	for index, header := range ExcelHeaders {
		cell, err := excelize.CoordinatesToCellName(index+1, 1)
		if err != nil {
			return err
		}
		if err := file.SetCellValue(excelSheetName, cell, header); err != nil {
			return err
		}
	}
	lastHeaderCell, err := excelize.CoordinatesToCellName(len(ExcelHeaders), 1)
	if err != nil {
		return err
	}
	headerStyle, err := file.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
		},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"D9EAF7"}, Pattern: 1},
		Border: []excelize.Border{
			{Type: "left", Color: "CCCCCC", Style: 1},
			{Type: "top", Color: "CCCCCC", Style: 1},
			{Type: "right", Color: "CCCCCC", Style: 1},
			{Type: "bottom", Color: "CCCCCC", Style: 1},
		},
	})
	if err != nil {
		return err
	}
	if err := file.SetCellStyle(excelSheetName, "A1", lastHeaderCell, headerStyle); err != nil {
		return err
	}
	if err := file.SetPanes(excelSheetName, &excelize.Panes{Freeze: true, Split: false, XSplit: 0, YSplit: 1, TopLeftCell: "A2", ActivePane: "bottomLeft"}); err != nil {
		return err
	}
	lastColumn, err := excelize.ColumnNumberToName(len(ExcelHeaders))
	if err != nil {
		return err
	}
	if err := file.SetColWidth(excelSheetName, "A", lastColumn, 18); err != nil {
		return err
	}

	return nil
}

func writeExcelRow(file *excelize.File, row int, transferID int, transfer models.Transferencia, item models.ItemTransferido) error {
	unit := item.Unidade
	values := []any{
		transferID,
		transfer.DataHora.Format("02/01/2006 15:04:05"),
		transfer.Usuario,
		transfer.Cargo,
		transfer.Solicitante,
		transfer.Observacao,
		transfer.ObraOrigemID,
		transfer.ObraOrigemNome,
		excelAppropriation(item.ApropriacaoOrigemCodigo, item.Apropriacao, item.ApropriacaoOrigemDescricao, item.ApropriacaoDescricao),
		excelQuantity(item.QuantidadeEstoqueOrigemAntes, unit),
		excelQuantity(quantityOrFallback(item.QuantidadeEnviada, item.Quantidade), unit),
		excelQuantity(item.QuantidadeEstoqueOrigemDepois, unit),
		excelOptionalQuantity(item.QuantidadeApropriacaoOrigemAntes, unit),
		excelOptionalQuantity(item.QuantidadeApropriacaoOrigemDepois, unit),
		transfer.ObraDestinoID,
		transfer.ObraDestinoNome,
		notApplicableString(item.ApropriacaoDestinoCodigo, item.ApropriacaoDestino),
		notApplicableString(item.ApropriacaoDestinoDescricaoSnapshot, item.ApropriacaoDestinoDescricao),
		excelQuantity(item.QuantidadeEstoqueDestinoAntes, unit),
		excelQuantity(quantityOrFallback(item.QuantidadeRecebida, item.Quantidade), unit),
		excelQuantity(item.QuantidadeEstoqueDestinoDepois, unit),
		excelOptionalQuantity(item.QuantidadeApropriacaoDestinoAntes, unit),
		excelOptionalQuantity(item.QuantidadeApropriacaoDestinoDepois, unit),
		item.ID,
		item.Nome,
		item.Detalhe,
		item.Marca,
		item.Unidade,
		models.TransferKindLabel(transfer.TransferKind),
		loanStatusForExcel(transfer),
	}

	for index, value := range values {
		cell, err := excelize.CoordinatesToCellName(index+1, row)
		if err != nil {
			return err
		}
		if err := file.SetCellValue(excelSheetName, cell, value); err != nil {
			return err
		}
	}

	if err := applyLoanStatusExcelStyle(file, row, loanStatusForExcel(transfer)); err != nil {
		return err
	}

	return nil
}

func applyLoanStatusExcelStyle(file *excelize.File, row int, status string) error {
	fillColor, ok := loanStatusExcelFillColor(status)
	if !ok {
		return nil
	}
	styleID, err := file.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{fillColor}, Pattern: 1},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
		},
	})
	if err != nil {
		return err
	}
	cell, err := excelize.CoordinatesToCellName(len(ExcelHeaders), row)
	if err != nil {
		return err
	}
	return file.SetCellStyle(excelSheetName, cell, cell, styleID)
}

func loanStatusExcelFillColor(status string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case strings.ToLower(models.LoanStatusLabel(models.LoanStatusPending)):
		return "FF0000", true
	case strings.ToLower(models.LoanStatusLabel(models.LoanStatusReturned)):
		return "00B050", true
	case strings.ToLower(models.LoanStatusLabel(models.LoanStatusPartiallyReturned)):
		return "0070C0", true
	default:
		return "", false
	}
}

func excelStyleHasFillColor(style *excelize.Style, color string) bool {
	if style == nil || style.Fill.Type != "pattern" || style.Fill.Pattern == 0 {
		return false
	}
	for _, current := range style.Fill.Color {
		if strings.EqualFold(current, color) {
			return true
		}
	}
	return false
}

func nextExcelTransferID(file *excelize.File) (int, error) {
	rows, err := file.GetRows(excelSheetName)
	if err != nil {
		return 0, err
	}
	maxID := 0
	for index, row := range rows {
		if index == 0 || len(row) == 0 {
			continue
		}
		id, err := strconv.Atoi(row[0])
		if err == nil && id > maxID {
			maxID = id
		}
	}
	return maxID + 1, nil
}

func loanStatusForExcel(transfer models.Transferencia) string {
	if models.EffectiveTransferKind(transfer.TransferKind) == models.TransferKindNotApplicable || transfer.LoanStatus == "" {
		return "Nao se aplica"
	}
	return models.LoanStatusLabel(transfer.LoanStatus)
}

func notApplicableString(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return "Nao se aplica"
}

func excelOptionalQuantity(value *float64, unit string) string {
	if value == nil {
		return "Nao se aplica"
	}
	return excelQuantity(*value, unit)
}

func excelQuantity(value float64, unit string) string {
	return models.FormatQuantidade(value, unit)
}

func excelAppropriation(codeValues ...string) string {
	code := notApplicableString(codeValues[0], codeValues[1])
	description := notApplicableString(codeValues[2], codeValues[3])
	if code == "Nao se aplica" {
		return code
	}
	if description == "Nao se aplica" {
		return code
	}
	return code + " - " + description
}

func quantityOrFallback(value float64, fallback float64) float64 {
	if value == 0 {
		return fallback
	}
	return value
}

func nextExcelRow(file *excelize.File) (int, error) {
	rows, err := file.GetRows(excelSheetName)
	if err != nil {
		return 0, err
	}
	if len(rows) == 0 {
		if err := writeExcelHeaders(file); err != nil {
			return 0, err
		}
		return 2, nil
	}

	return len(rows) + 1, nil
}

func readExcelCell(file *excelize.File, row, column int) (string, error) {
	cell, err := excelize.CoordinatesToCellName(column, row)
	if err != nil {
		return "", err
	}

	return file.GetCellValue(excelSheetName, cell)
}

func excelFloatString(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}
