package storage

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"

	"github.com/xuri/excelize/v2"

	"sienge-transfer/models"
)

const (
	TransferenciasExcelFileName = "transferencias.xlsx"
	excelSheetName              = "Transferencias"
)

var ExcelHeaders = []string{
	"ID Movimento Sienge",
	"Data/Hora",
	"Usuario",
	"Cargo",
	"Solicitante",
	"Observacao",
	"Codigo Tipo Documento",
	"Codigo Tipo Movimento",
	"Obra Origem ID",
	"Obra Origem Nome",
	"Apropriacao Origem Codigo",
	"Apropriacao Origem Descricao",
	"Apropriacao Origem BuildingUnitID",
	"Apropriacao Origem SheetItemID",
	"Quantidade Origem no Momento da Transferencia",
	"Quantidade Enviada",
	"Quantidade Origem Apos Transferencia",
	"Quantidade Apropriacao Origem no Momento da Transferencia",
	"Quantidade Apropriacao Origem Apos Transferencia",
	"Obra Destino ID",
	"Obra Destino Nome",
	"Apropriacao Destino Codigo",
	"Apropriacao Destino Descricao",
	"Apropriacao Destino BuildingUnitID",
	"Apropriacao Destino SheetItemID",
	"Quantidade Destino no Momento da Transferencia",
	"Quantidade Recebida",
	"Quantidade Destino Apos Transferencia",
	"Quantidade Apropriacao Destino no Momento da Transferencia",
	"Quantidade Apropriacao Destino Apos Transferencia",
	"Insumo ID",
	"Nome do Insumo",
	"Detalhe",
	"Detalhe ID",
	"Marca",
	"Marca ID",
	"Unidade",
	"Preco Unitario",
	"Tipo da Transferencia",
	"ID do Emprestimo",
	"Status do Emprestimo",
	"E Devolucao de Emprestimo",
}

func (s Store) EnsureExcelFromHistory() error {
	_, err := os.Stat(s.ExcelPath())
	if err == nil {
		return nil
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
	for _, transfer := range history {
		for _, item := range transfer.Insumos {
			if err := writeExcelRow(file, nextRow, transfer, item); err != nil {
				return err
			}
			nextRow++
		}
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
	for _, item := range transfer.Insumos {
		if err := writeExcelRow(file, nextRow, transfer, item); err != nil {
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

func writeExcelRow(file *excelize.File, row int, transfer models.Transferencia, item models.ItemTransferido) error {
	values := []any{
		transfer.IDMovimento,
		transfer.DataHora.Format("02/01/2006 15:04:05"),
		transfer.Usuario,
		transfer.Cargo,
		transfer.Solicitante,
		transfer.Observacao,
		transfer.CodigoTipoDocumento,
		transfer.CodigoTipoMovimento,
		transfer.ObraOrigemID,
		transfer.ObraOrigemNome,
		notApplicableString(item.ApropriacaoOrigemCodigo, item.Apropriacao),
		notApplicableString(item.ApropriacaoOrigemDescricao, item.ApropriacaoDescricao),
		notApplicableInt(item.ApropriacaoOrigemBuildingUnitID),
		notApplicableInt(item.ApropriacaoOrigemSheetItemID),
		item.QuantidadeEstoqueOrigemAntes,
		quantityOrFallback(item.QuantidadeEnviada, item.Quantidade),
		item.QuantidadeEstoqueOrigemDepois,
		notApplicableFloat(item.QuantidadeApropriacaoOrigemAntes),
		notApplicableFloat(item.QuantidadeApropriacaoOrigemDepois),
		transfer.ObraDestinoID,
		transfer.ObraDestinoNome,
		notApplicableString(item.ApropriacaoDestinoCodigo, item.ApropriacaoDestino),
		notApplicableString(item.ApropriacaoDestinoDescricaoSnapshot, item.ApropriacaoDestinoDescricao),
		notApplicableInt(item.ApropriacaoDestinoBuildingUnitID),
		notApplicableInt(item.ApropriacaoDestinoSheetItemID),
		item.QuantidadeEstoqueDestinoAntes,
		quantityOrFallback(item.QuantidadeRecebida, item.Quantidade),
		item.QuantidadeEstoqueDestinoDepois,
		notApplicableFloat(item.QuantidadeApropriacaoDestinoAntes),
		notApplicableFloat(item.QuantidadeApropriacaoDestinoDepois),
		item.ID,
		item.Nome,
		item.Detalhe,
		item.DetalheID,
		item.Marca,
		item.MarcaID,
		item.Unidade,
		item.PrecoUnitario,
		models.TransferKindLabel(transfer.TransferKind),
		transfer.LinkedLoanID,
		loanStatusForExcel(transfer),
		models.EffectiveTransferKind(transfer.TransferKind) == models.TransferKindReturn && transfer.LinkedLoanID != "",
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

	return nil
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

func notApplicableInt(value int) any {
	if value <= 0 {
		return "Nao se aplica"
	}
	return value
}

func notApplicableFloat(value *float64) any {
	if value == nil {
		return "Nao se aplica"
	}
	return *value
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
