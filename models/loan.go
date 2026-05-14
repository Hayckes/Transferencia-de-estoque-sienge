package models

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

const quantityEpsilon = 0.0001

type TransferKind string

const (
	TransferKindLoan          TransferKind = "loan"
	TransferKindReturn        TransferKind = "return"
	TransferKindNotApplicable TransferKind = "not_applicable"
)

type LoanStatus string

const (
	LoanStatusPending           LoanStatus = "pending"
	LoanStatusPartiallyReturned LoanStatus = "partially_returned"
	LoanStatusReturned          LoanStatus = "returned"
)

type LoanRecord struct {
	ID                    string       `json:"id"`
	OriginalTransferID    string       `json:"original_transfer_id,omitempty"`
	OriginalMovementID    string       `json:"original_movement_id,omitempty"`
	ReturnTransferIDs     []string     `json:"return_transfer_ids,omitempty"`
	ReturnMovementIDs     []string     `json:"return_movement_ids,omitempty"`
	CreatedAt             time.Time    `json:"created_at"`
	LoanDate              time.Time    `json:"loan_date"`
	LastReturnDate        *time.Time   `json:"last_return_date,omitempty"`
	OriginWorkID          int          `json:"origin_work_id"`
	OriginWorkName        string       `json:"origin_work_name"`
	DestinationWorkID     int          `json:"destination_work_id"`
	DestinationWorkName   string       `json:"destination_work_name"`
	Solicitor             string       `json:"solicitor"`
	User                  string       `json:"user"`
	Role                  string       `json:"role"`
	Observation           string       `json:"observation,omitempty"`
	Type                  TransferKind `json:"type"`
	Status                LoanStatus   `json:"status"`
	Items                 []LoanItem   `json:"items"`
	TotalLoanedQuantity   float64      `json:"total_loaned_quantity"`
	TotalReturnedQuantity float64      `json:"total_returned_quantity"`
	ItemCount             int          `json:"item_count"`
}

type LoanItem struct {
	ResourceID                          int     `json:"resource_id"`
	ResourceName                        string  `json:"resource_name"`
	DetailID                            *int    `json:"detail_id,omitempty"`
	DetailName                          string  `json:"detail_name,omitempty"`
	BrandID                             *int    `json:"brand_id,omitempty"`
	BrandName                           string  `json:"brand_name,omitempty"`
	Unit                                string  `json:"unit,omitempty"`
	UnitPrice                           float64 `json:"unit_price,omitempty"`
	LoanedQuantity                      float64 `json:"loaned_quantity"`
	ReturnedQuantity                    float64 `json:"returned_quantity"`
	OriginAppropriationCode             string  `json:"origin_appropriation_code,omitempty"`
	OriginAppropriationDescription      string  `json:"origin_appropriation_description,omitempty"`
	OriginBuildingUnitID                *int    `json:"origin_building_unit_id,omitempty"`
	OriginSheetItemID                   *int    `json:"origin_sheet_item_id,omitempty"`
	DestinationAppropriationCode        string  `json:"destination_appropriation_code,omitempty"`
	DestinationAppropriationDescription string  `json:"destination_appropriation_description,omitempty"`
	DestinationBuildingUnitID           *int    `json:"destination_building_unit_id,omitempty"`
	DestinationSheetItemID              *int    `json:"destination_sheet_item_id,omitempty"`
}

func EffectiveTransferKind(kind TransferKind) TransferKind {
	switch kind {
	case TransferKindLoan, TransferKindReturn, TransferKindNotApplicable:
		return kind
	default:
		return TransferKindNotApplicable
	}
}

func TransferKindLabel(kind TransferKind) string {
	switch EffectiveTransferKind(kind) {
	case TransferKindLoan:
		return "Emprestimo"
	case TransferKindReturn:
		return "Devolucao"
	default:
		return "Nao se aplica"
	}
}

func TransferKindFromLabel(label string) TransferKind {
	switch strings.ToLower(strings.TrimSpace(label)) {
	case "emprestimo":
		return TransferKindLoan
	case "devolucao":
		return TransferKindReturn
	default:
		return TransferKindNotApplicable
	}
}

func LoanStatusLabel(status LoanStatus) string {
	switch status {
	case LoanStatusPartiallyReturned:
		return "Parcialmente devolvido"
	case LoanStatusReturned:
		return "Devolvido"
	default:
		return "Pendente"
	}
}

func (i LoanItem) PendingQuantity() float64 {
	pending := i.LoanedQuantity - i.ReturnedQuantity
	if pending < quantityEpsilon {
		return 0
	}
	return pending
}

func (l *LoanRecord) Recalculate() {
	loaned := 0.0
	returned := 0.0
	anyReturned := false
	anyPending := false
	for _, item := range l.Items {
		loaned += item.LoanedQuantity
		returned += item.ReturnedQuantity
		if item.ReturnedQuantity > quantityEpsilon {
			anyReturned = true
		}
		if item.PendingQuantity() > quantityEpsilon {
			anyPending = true
		}
	}
	l.TotalLoanedQuantity = loaned
	l.TotalReturnedQuantity = returned
	l.ItemCount = len(l.Items)
	switch {
	case !anyPending && len(l.Items) > 0:
		l.Status = LoanStatusReturned
	case anyReturned:
		l.Status = LoanStatusPartiallyReturned
	default:
		l.Status = LoanStatusPending
	}
}

func (l LoanRecord) PendingItems() []LoanItem {
	items := make([]LoanItem, 0, len(l.Items))
	for _, item := range l.Items {
		if item.PendingQuantity() > quantityEpsilon {
			items = append(items, item)
		}
	}
	return items
}

func CreateLoanRecordFromTransfer(transfer Transferencia) LoanRecord {
	record := LoanRecord{
		ID:                  BuildLoanID(transfer),
		OriginalMovementID:  transfer.IDMovimento,
		CreatedAt:           time.Now(),
		LoanDate:            transfer.DataHora,
		OriginWorkID:        transfer.ObraOrigemID,
		OriginWorkName:      transfer.ObraOrigemNome,
		DestinationWorkID:   transfer.ObraDestinoID,
		DestinationWorkName: transfer.ObraDestinoNome,
		Solicitor:           transfer.Solicitante,
		User:                transfer.Usuario,
		Role:                transfer.Cargo,
		Observation:         transfer.Observacao,
		Type:                TransferKindLoan,
		Status:              LoanStatusPending,
		Items:               make([]LoanItem, 0, len(transfer.Insumos)),
	}
	for _, item := range transfer.Insumos {
		record.Items = append(record.Items, LoanItemFromTransferItem(item))
	}
	record.Recalculate()
	return record
}

func BuildLoanID(transfer Transferencia) string {
	movementID := strings.TrimSpace(transfer.IDMovimento)
	if movementID == "" {
		movementID = "sem-movimento"
	}
	return fmt.Sprintf("loan-%d-%s", transfer.DataHora.UnixNano(), sanitizeLoanIDPart(movementID))
}

func LoanItemFromTransferItem(item ItemTransferido) LoanItem {
	return LoanItem{
		ResourceID:                          item.ID,
		ResourceName:                        item.Nome,
		DetailID:                            positiveIntPtr(item.DetalheID),
		DetailName:                          item.Detalhe,
		BrandID:                             positiveIntPtr(item.MarcaID),
		BrandName:                           item.Marca,
		Unit:                                item.Unidade,
		UnitPrice:                           item.PrecoUnitario,
		LoanedQuantity:                      quantityOrFallback(item.QuantidadeEnviada, item.Quantidade),
		OriginAppropriationCode:             firstNonEmpty(item.ApropriacaoOrigemCodigo, item.Apropriacao),
		OriginAppropriationDescription:      firstNonEmpty(item.ApropriacaoOrigemDescricao, item.ApropriacaoDescricao),
		OriginBuildingUnitID:                positiveIntPtr(item.ApropriacaoOrigemBuildingUnitID),
		OriginSheetItemID:                   positiveIntPtr(item.ApropriacaoOrigemSheetItemID),
		DestinationAppropriationCode:        firstNonEmpty(item.ApropriacaoDestinoCodigo, item.ApropriacaoDestino),
		DestinationAppropriationDescription: firstNonEmpty(item.ApropriacaoDestinoDescricaoSnapshot, item.ApropriacaoDestinoDescricao),
		DestinationBuildingUnitID:           positiveIntPtr(item.ApropriacaoDestinoBuildingUnitID),
		DestinationSheetItemID:              positiveIntPtr(item.ApropriacaoDestinoSheetItemID),
	}
}

func ApplyReturnToLoan(loan LoanRecord, transfer Transferencia) (LoanRecord, error) {
	loan.Items = append([]LoanItem(nil), loan.Items...)
	loan.ReturnTransferIDs = append([]string(nil), loan.ReturnTransferIDs...)
	loan.ReturnMovementIDs = append([]string(nil), loan.ReturnMovementIDs...)
	for _, returnedItem := range transfer.Insumos {
		matched := false
		for index := range loan.Items {
			if !loanItemMatchesTransferItem(loan.Items[index], returnedItem) {
				continue
			}
			matched = true
			quantity := quantityOrFallback(returnedItem.QuantidadeRecebida, returnedItem.Quantidade)
			if quantity <= quantityEpsilon {
				return LoanRecord{}, fmt.Errorf("a quantidade de devolucao do item %d deve ser maior que zero", returnedItem.ID)
			}
			pending := loan.Items[index].PendingQuantity()
			if quantity-pending > quantityEpsilon {
				return LoanRecord{}, fmt.Errorf("a quantidade de devolucao do item %d e maior que a quantidade pendente do emprestimo. Quantidade pendente: %.4f. Quantidade informada: %.4f", returnedItem.ID, pending, quantity)
			}
			loan.Items[index].ReturnedQuantity += quantity
			break
		}
		if !matched {
			return LoanRecord{}, fmt.Errorf("insumo %d nao pertence ao emprestimo vinculado", returnedItem.ID)
		}
	}
	now := transfer.DataHora
	loan.LastReturnDate = &now
	if strings.TrimSpace(transfer.LinkedLoanID) != "" {
		loan.ReturnTransferIDs = appendIfMissing(loan.ReturnTransferIDs, transfer.LinkedLoanID)
	}
	if strings.TrimSpace(transfer.IDMovimento) != "" {
		loan.ReturnMovementIDs = appendIfMissing(loan.ReturnMovementIDs, transfer.IDMovimento)
	}
	loan.Recalculate()
	return loan, nil
}

func ValidateReturnAgainstLoan(loan LoanRecord, transfer Transferencia) error {
	if EffectiveTransferKind(transfer.TransferKind) != TransferKindReturn || strings.TrimSpace(transfer.LinkedLoanID) == "" {
		return nil
	}
	_, err := ApplyReturnToLoan(loan, transfer)
	return err
}

func loanItemMatchesTransferItem(loanItem LoanItem, transferItem ItemTransferido) bool {
	return loanItem.ResourceID == transferItem.ID && ptrValue(loanItem.DetailID) == transferItem.DetalheID && ptrValue(loanItem.BrandID) == transferItem.MarcaID
}

func CanReturnLoan(status LoanStatus) bool {
	return status == LoanStatusPending || status == LoanStatusPartiallyReturned
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func quantityOrFallback(value float64, fallback float64) float64 {
	if value == 0 {
		return fallback
	}
	return value
}

func positiveIntPtr(value int) *int {
	if value <= 0 {
		return nil
	}
	return &value
}

func ptrValue(value *int) int {
	if value == nil {
		return 0
	}
	return *value
}

func appendIfMissing(values []string, value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return values
	}
	for _, current := range values {
		if current == value {
			return values
		}
	}
	return append(values, value)
}

func sanitizeLoanIDPart(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return "sem-id"
	}
	var builder strings.Builder
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			builder.WriteRune(r)
			continue
		}
		builder.WriteRune('-')
	}
	return strings.Trim(builder.String(), "-")
}

var ErrLoanNotFound = errors.New("emprestimo nao encontrado")
