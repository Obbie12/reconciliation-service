package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
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
	BatchID   string                    `json:"reconciliation_id"`
	Status    string                    `json:"status"`
	Matches   []*matching.MatchesResult `json:"matches"`
	Unmatched []*matching.UnmatchResult `json:"unmatched,omitempty"`
	Summary   map[string]interface{}    `json:"summary"`
}

func (s *ReconciliationService) GetBankTransactions(fromDate, toDate string) ([]*models.BankTransaction, error) {
	return s.bankRepo.GetUnreconciledTransactions(fromDate, toDate)
}

func (s *ReconciliationService) GetAccountingEntries(fromDate, toDate string) ([]*models.AccountingEntry, error) {
	return s.accountingRepo.GetUnreconciledEntries(fromDate, toDate)
}

func (s *ReconciliationService) StartReconciliation(fromDate, toDate string) (*ReconciliationResult, error) {
	bankTransactions, err := s.bankRepo.GetUnreconciledTransactions(fromDate, toDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get unreconciled bank transactions: %v", err)
	}

	accountingEntries, err := s.accountingRepo.GetUnreconciledEntries(fromDate, toDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get unreconciled accounting entries: %v", err)
	}

	return s.ProcessReconciliationWithData(fromDate, toDate, bankTransactions, accountingEntries)
}

func (s *ReconciliationService) ProcessReconciliationWithData(fromDate, toDate string, bankTransactions []*models.BankTransaction, accountingEntries []*models.AccountingEntry) (*ReconciliationResult, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	batchID := fmt.Sprintf("REC-%s", time.Now().Format("20060102-150405"))

	s.matchEngine.SetData(bankTransactions, accountingEntries)

	matchChan := make(chan []*matching.MatchResult, 1)
	matchErrChan := make(chan error, 1)

	go func() {
		matches, err := s.matchEngine.ProcessMatches()
		if err != nil {
			matchErrChan <- fmt.Errorf("failed to process matches: %v", err)
			return
		}
		matchChan <- matches
	}()

	var matches []*matching.MatchResult
	select {
	case err := <-matchErrChan:
		return nil, err
	case matches = <-matchChan:
	}

	type processResult struct {
		bankIDs       map[int64]bool
		accountingIDs map[int64]bool
		err           error
	}
	processChan := make(chan processResult, len(matches))

	var wg sync.WaitGroup
	for _, match := range matches {
		wg.Add(1)
		go func(m *matching.MatchResult) {
			defer wg.Done()

			result := processResult{
				bankIDs:       make(map[int64]bool),
				accountingIDs: make(map[int64]bool),
			}

			reconciliation := &models.Reconciliation{
				BatchID:          batchID,
				Status:           "matched",
				MatchConfidence:  m.Confidence,
				AmountDifference: m.AmountDifference,
			}

			err := s.reconciliationRepo.CreateReconciliation(tx, reconciliation)
			if err != nil {
				result.err = fmt.Errorf("failed to create reconciliation batch: %v", err)
				processChan <- result
				return
			}

			mapping := &models.ReconciliationMapping{
				ReconciliationID: reconciliation.ID,
				BankTransactionID: sql.NullInt64{
					Int64: m.BankTransaction.ID,
					Valid: true,
				},
				MappingType: m.Type,
			}

			if m.Type == models.MappingOneToOne {
				mapping.AccountingEntryID = sql.NullInt64{
					Int64: m.AccountingEntries[0].ID,
					Valid: true,
				}
				err = s.reconciliationRepo.CreateMapping(tx, mapping)
				if err != nil {
					result.err = fmt.Errorf("failed to create mapping: %v", err)
					processChan <- result
					return
				}

				result.bankIDs[m.BankTransaction.ID] = true
				result.accountingIDs[m.AccountingEntries[0].ID] = true
			} else {
				for _, ae := range m.AccountingEntries {
					mapping.AccountingEntryID = sql.NullInt64{
						Int64: ae.ID,
						Valid: true,
					}
					err = s.reconciliationRepo.CreateMapping(tx, mapping)
					if err != nil {
						result.err = fmt.Errorf("failed to create mapping: %v", err)
						processChan <- result
						return
					}
					result.accountingIDs[ae.ID] = true
				}
				result.bankIDs[m.BankTransaction.ID] = true
			}

			auditDetails, _ := json.Marshal(map[string]interface{}{
				"match_type":     m.Type,
				"confidence":     m.Confidence,
				"match_criteria": m.MatchCriteria,
			})

			audit := &models.ReconciliationAudit{
				ReconciliationID: reconciliation.ID,
				Action:           models.AuditActionMatched,
				Details:          auditDetails,
			}
			err = s.reconciliationRepo.CreateAuditEntry(tx, audit)
			if err != nil {
				result.err = fmt.Errorf("failed to create audit entry: %v", err)
				processChan <- result
				return
			}

			processChan <- result
		}(match)
	}

	go func() {
		wg.Wait()
		close(processChan)
	}()

	processedBankIDs := make(map[int64]bool)
	processedAccountingIDs := make(map[int64]bool)

	for result := range processChan {
		if result.err != nil {
			return nil, result.err
		}
		for id := range result.bankIDs {
			processedBankIDs[id] = true
		}
		for id := range result.accountingIDs {
			processedAccountingIDs[id] = true
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

	summary := map[string]interface{}{
		"total_processed": len(bankTransactions) + len(accountingEntries),
		"matched":         len(matches),
		"unmatched":       len(unmatchedBank),
		"disputed":        0,
	}

	var m []*matching.MatchesResult
	for _, match := range matches {
		var entryIDs []string
		for _, ae := range match.AccountingEntries {
			entryIDs = append(entryIDs, ae.EntryID)
		}

		data := matching.MatchesResult{
			Type:             match.Type,
			Confidence:       match.Confidence,
			BankTransaction:  match.BankTransaction.TransactionID,
			AccountingEntry:  fmt.Sprintf("%v", entryIDs),
			AmountDifference: match.AmountDifference,
			MatchCriteria:    match.MatchCriteria,
		}
		m = append(m, &data)
	}

	var um []*matching.UnmatchResult
	for _, unmatch := range unmatchedAccounting {
		var entryIDs []string
		var trID string
		entryIDs = append(entryIDs, unmatch.EntryID)

		invoiceMap := make(map[string]struct{})
		invoiceMap[unmatch.InvoiceNumber] = struct{}{}

		for _, transaction := range bankTransactions {
			if _, exists := invoiceMap[transaction.ReferenceNumber]; exists {
				trID = transaction.TransactionID
			}
		}

		data := matching.UnmatchResult{
			BankTransactions:  trID,
			AccountingEntries: entryIDs,
		}

		reconciliation := &models.Reconciliation{
			BatchID:          batchID,
			Status:           "unmatched",
			MatchConfidence:  0,
			AmountDifference: 0,
		}
		err = s.reconciliationRepo.CreateReconciliation(tx, reconciliation)
		if err != nil {
			return nil, fmt.Errorf("failed to create reconciliation batch: %v", err)
		}

		auditDetails, _ := json.Marshal(map[string]interface{}{
			"bank_transactions":  trID,
			"accounting_entries": entryIDs,
		})

		audit := &models.ReconciliationAudit{
			ReconciliationID: reconciliation.ID,
			Action:           models.AuditActionUnmatched,
			Details:          auditDetails,
		}
		err = s.reconciliationRepo.CreateAuditEntry(tx, audit)
		if err != nil {
			return nil, fmt.Errorf("failed to create audit entry: %v", err)
		}

		um = append(um, &data)
	}

	// Commit transaction
	err = tx.Commit()
	if err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %v", err)
	}

	var status string
	if len(um) > 0 {
		status = "completed"
	} else {
		status = "matches"
	}

	return &ReconciliationResult{
		BatchID:   batchID,
		Status:    status,
		Matches:   m,
		Unmatched: um,
		Summary:   summary,
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
