package storage

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"sienge-transfer/models"
)

const HistoricoFileName = "historico.json"

type Store struct {
	Dir string
}

type HistoricoResumo struct {
	DataHora        string
	IDMovimento     string
	Solicitante     string
	ObraOrigem      string
	ObraDestino     string
	QuantidadeItens int
	TotalQuantidade float64
}

func NewStore(dir string) Store {
	return Store{Dir: dir}
}

func (s Store) EnsureDir() error {
	return os.MkdirAll(s.Dir, 0o700)
}

func (s Store) HistoricoPath() string {
	return filepath.Join(s.Dir, HistoricoFileName)
}

func (s Store) ExcelPath() string {
	return filepath.Join(s.Dir, TransferenciasExcelFileName)
}

func (s Store) EnsureHistory() error {
	if err := s.EnsureDir(); err != nil {
		return err
	}

	_, err := os.Stat(s.HistoricoPath())
	if err == nil {
		return nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	return os.WriteFile(s.HistoricoPath(), []byte("[]\n"), 0o600)
}

func (s Store) ReadHistory() ([]models.Transferencia, error) {
	if err := s.EnsureHistory(); err != nil {
		return nil, err
	}

	data, err := os.ReadFile(s.HistoricoPath())
	if err != nil {
		return nil, err
	}

	var history []models.Transferencia
	if err := json.Unmarshal(data, &history); err != nil {
		return nil, err
	}

	if history == nil {
		return []models.Transferencia{}, nil
	}

	return history, nil
}

func (s Store) WriteHistory(history []models.Transferencia) error {
	if err := s.EnsureDir(); err != nil {
		return err
	}
	if history == nil {
		history = []models.Transferencia{}
	}

	data, err := json.MarshalIndent(history, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	return os.WriteFile(s.HistoricoPath(), data, 0o600)
}

func (s Store) AppendHistory(transfer models.Transferencia) error {
	history, err := s.ReadHistory()
	if err != nil {
		return err
	}

	history = append(history, transfer)
	return s.WriteHistory(history)
}

func (s Store) ReadHistorySummary() ([]HistoricoResumo, error) {
	history, err := s.ReadHistory()
	if err != nil {
		return nil, err
	}

	summaries := make([]HistoricoResumo, 0, len(history))
	for _, transfer := range history {
		summaries = append(summaries, SummarizeTransfer(transfer))
	}

	return summaries, nil
}

func SummarizeTransfer(transfer models.Transferencia) HistoricoResumo {
	total := 0.0
	for _, item := range transfer.Insumos {
		total += item.Quantidade
	}

	return HistoricoResumo{
		DataHora:        transfer.DataHora.Format("02/01/2006 15:04:05"),
		IDMovimento:     transfer.IDMovimento,
		Solicitante:     transfer.Solicitante,
		ObraOrigem:      models.Obra{ID: transfer.ObraOrigemID, Nome: transfer.ObraOrigemNome}.Label(),
		ObraDestino:     models.Obra{ID: transfer.ObraDestinoID, Nome: transfer.ObraDestinoNome}.Label(),
		QuantidadeItens: len(transfer.Insumos),
		TotalQuantidade: total,
	}
}
