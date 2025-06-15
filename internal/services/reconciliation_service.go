package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"reconciliation-service/internal/matching"
	"reconciliation-service/internal/models"
	"reconciliation-service/internal/repositories"
)

type ReconciliationService struct {
	db                 *sql.DB
	matchEngine        *matching.MatchEngine
	bankRepo           repositories.BankRepository
	accountingRepo     repositories.AccountingRepository
	reconciliationRepo repositories.ReconciliationRepository
}

func NewReconciliationService(
	db *sql.DB,
	bankRepo repositories.BankRepository,
	accountingRepo repositories.AccountingRepository,
	reconciliationRepo repositories.ReconciliationRepository,
) *ReconciliationService {
	return &ReconciliationService{
		db:                 db,
		matchEngine:        matching.NewMatchEngine(),
		bankRepo:           bankRepo,
		accountingRepo:     accountingRepo,
		reconciliationRepo: reconciliationRepo,
	}
}

type ReconciliationResult struct {
	BatchID             string                    `json:"batch_id"`
	Status              string                    `json:"status"`
	Matches             []*matching.MatchResult   `json:"matches"`
	UnmatchedBank       []*models.BankTransaction `json:"unmatched_bank,omitempty"`
	UnmatchedAccounting []*models.AccountingEntry `json:"unmatched_accounting,omitempty"`
	Summary             map[string]interface{}    `json:"summary"`
}

func (s *ReconciliationService) StartReconciliation(fromDate, toDate string) (*ReconciliationResult, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	batchID := fmt.Sprintf("REC-%s", time.Now().Format("20060102-150405"))
	reconciliation := &models.Reconciliation{
		BatchID: batchID,
		Status:  models.StatusMatched,
	}

	err = s.reconciliationRepo.CreateReconciliation(tx, reconciliation)
	if err != nil {
		return nil, fmt.Errorf("failed to create reconciliation batch: %v", err)
	}

	bankTransactions, err := s.bankRepo.GetUnreconciledTransactions(fromDate, toDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get unreconciled bank transactions: %v", err)
	}

	accountingEntries, err := s.accountingRepo.GetUnreconciledEntries(fromDate, toDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get unreconciled accounting entries: %v", err)
	}

	s.matchEngine.SetData(bankTransactions, accountingEntries)

	matches, err := s.matchEngine.ProcessMatches()
	if err != nil {
		return nil, fmt.Errorf("failed to process matches: %v", err)
	}

	var processedBankIDs = make(map[int64]bool)
	var processedAccountingIDs = make(map[int64]bool)

	for _, match := range matches {
		mapping := &models.ReconciliationMapping{
			ReconciliationID: reconciliation.ID,
			BankTransactionID: sql.NullInt64{
				Int64: match.BankTransaction.ID,
				Valid: true,
			},
			MappingType: match.Type,
		}

		if match.Type == models.MappingOneToOne {
			mapping.AccountingEntryID = sql.NullInt64{
				Int64: match.AccountingEntries[0].ID,
				Valid: true,
			}
			err = s.reconciliationRepo.CreateMapping(tx, mapping)
			if err != nil {
				return nil, fmt.Errorf("failed to create mapping: %v", err)
			}

			processedBankIDs[match.BankTransaction.ID] = true
			processedAccountingIDs[match.AccountingEntries[0].ID] = true
		} else {
			for _, ae := range match.AccountingEntries {
				mapping.AccountingEntryID = sql.NullInt64{
					Int64: ae.ID,
					Valid: true,
				}
				err = s.reconciliationRepo.CreateMapping(tx, mapping)
				if err != nil {
					return nil, fmt.Errorf("failed to create mapping: %v", err)
				}
				processedAccountingIDs[ae.ID] = true
			}
			processedBankIDs[match.BankTransaction.ID] = true
		}
		auditDetails, _ := json.Marshal(map[string]interface{}{
			"match_type":     match.Type,
			"confidence":     match.Confidence,
			"match_criteria": match.MatchCriteria,
		})

		audit := &models.ReconciliationAudit{
			ReconciliationID: reconciliation.ID,
			Action:           models.AuditActionMatched,
			Details:          auditDetails,
		}
		err = s.reconciliationRepo.CreateAuditEntry(tx, audit)
		if err != nil {
			return nil, fmt.Errorf("failed to create audit entry: %v", err)
		}
	}

	var unmatchedBank []*models.BankTransaction
	var unmatchedAccounting []*models.AccountingEntry

	for _, bt := range bankTransactions {
		if !processedBankIDs[bt.ID] {
			unmatchedBank = append(unmatchedBank, bt)
		}
	}

	for _, ae := range accountingEntries {
		if !processedAccountingIDs[ae.ID] {
			unmatchedAccounting = append(unmatchedAccounting, ae)
		}
	}

	if len(unmatchedBank) > 0 || len(unmatchedAccounting) > 0 {
		reconciliation.Status = models.StatusUnmatchedBank
		err = s.reconciliationRepo.UpdateReconciliationStatus(tx, reconciliation.ID, reconciliation.Status)
		if err != nil {
			return nil, fmt.Errorf("failed to update reconciliation status: %v", err)
		}
	}

	err = tx.Commit()
	if err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %v", err)
	}

	summary := map[string]interface{}{
		"total_processed":      len(bankTransactions) + len(accountingEntries),
		"matched":              len(matches),
		"unmatched_bank":       len(unmatchedBank),
		"unmatched_accounting": len(unmatchedAccounting),
	}

	return &ReconciliationResult{
		BatchID:             batchID,
		Status:              reconciliation.Status,
		Matches:             matches,
		UnmatchedBank:       unmatchedBank,
		UnmatchedAccounting: unmatchedAccounting,
		Summary:             summary,
	}, nil
}

func (s *ReconciliationService) GetReconciliationStatus(batchID string) (*ReconciliationResult, error) {
	reconciliation, err := s.reconciliationRepo.GetReconciliationByBatchID(batchID)
	if err != nil {
		return nil, fmt.Errorf("failed to get reconciliation: %v", err)
	}

	return &ReconciliationResult{
		BatchID: reconciliation.BatchID,
		Status:  reconciliation.Status,
	}, nil
}

func (s *ReconciliationService) ResolveDispute(batchID string, resolution map[string]interface{}) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	reconciliation, err := s.reconciliationRepo.GetReconciliationByBatchID(batchID)
	if err != nil {
		return fmt.Errorf("failed to get reconciliation: %v", err)
	}

	err = s.reconciliationRepo.UpdateReconciliationStatus(tx, reconciliation.ID, models.StatusMatched)
	if err != nil {
		return fmt.Errorf("failed to update reconciliation status: %v", err)
	}

	resolutionDetails, _ := json.Marshal(resolution)
	audit := &models.ReconciliationAudit{
		ReconciliationID: reconciliation.ID,
		Action:           models.AuditActionResolved,
		Details:          resolutionDetails,
	}
	err = s.reconciliationRepo.CreateAuditEntry(tx, audit)
	if err != nil {
		return fmt.Errorf("failed to create audit entry: %v", err)
	}

	return tx.Commit()
}

func (s *ReconciliationService) GetUnmatchedRecords(fromDate, toDate string) (map[string]interface{}, error) {
	return s.reconciliationRepo.GetUnmatchedRecords(fromDate, toDate)
}
