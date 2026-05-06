package storage

import (
	"errors"
	"os"
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
	"Data e Hora",
	"Usuario",
	"Cargo",
	"Solicitante",
	"Obra Origem",
	"Obra Destino",
	"ID Insumo",
	"Nome Insumo",
	"Detalhe",
	"Marca",
	"Apropriacao",
	"Quantidade",
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

	return file.SaveAs(s.ExcelPath())
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

	return file.SaveAs(s.ExcelPath())
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

	return nil
}

func writeExcelRow(file *excelize.File, row int, transfer models.Transferencia, item models.ItemTransferido) error {
	values := []any{
		transfer.IDMovimento,
		transfer.DataHora.Format("02/01/2006 15:04:05"),
		transfer.Usuario,
		transfer.Cargo,
		transfer.Solicitante,
		models.Obra{ID: transfer.ObraOrigemID, Nome: transfer.ObraOrigemNome}.Label(),
		models.Obra{ID: transfer.ObraDestinoID, Nome: transfer.ObraDestinoNome}.Label(),
		item.ID,
		item.Nome,
		item.Detalhe,
		item.Marca,
		item.Apropriacao,
		item.Quantidade,
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
