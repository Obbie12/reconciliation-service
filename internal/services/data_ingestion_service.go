package services

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"reconciliation-service/internal/models"
	"reconciliation-service/internal/repositories"
)

type DataIngestionService struct {
	db                 *sql.DB
	bankRepo           repositories.BankRepository
	accountingRepo     repositories.AccountingRepository
	reconciliationRepo repositories.ReconciliationRepository
}

func NewDataIngestionService(
	db *sql.DB,
	bankRepo repositories.BankRepository,
	accountingRepo repositories.AccountingRepository,
	reconciliationRepo repositories.ReconciliationRepository,
) *DataIngestionService {
	return &DataIngestionService{
		db:                 db,
		bankRepo:           bankRepo,
		accountingRepo:     accountingRepo,
		reconciliationRepo: reconciliationRepo,
	}
}

type BankTransactionInput struct {
	TransactionID   string  `json:"transaction_id"`
	AccountNumber   string  `json:"account_number"`
	Amount          float64 `json:"amount"`
	TransactionDate string  `json:"transaction_date"`
	Description     string  `json:"description,omitempty"`
	ReferenceNumber string  `json:"reference_number,omitempty"`
}

type AccountingEntryInput struct {
	EntryID       string  `json:"entry_id"`
	AccountCode   string  `json:"account_code"`
	Amount        float64 `json:"amount"`
	EntryDate     string  `json:"entry_date"`
	Description   string  `json:"description,omitempty"`
	InvoiceNumber string  `json:"invoice_number,omitempty"`
}

type IngestionResult struct {
	Success      bool                   `json:"success"`
	RecordsCount int                    `json:"records_count"`
	Errors       []string               `json:"errors,omitempty"`
	Details      map[string]interface{} `json:"details,omitempty"`
}

func (s *DataIngestionService) IngestBankTransactions(transactions []BankTransactionInput) (*IngestionResult, error) {
	result := &IngestionResult{
		Success: true,
		Details: make(map[string]interface{}),
	}

	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	for _, input := range transactions {
		if err := validateBankTransaction(input); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Invalid transaction %s: %v", input.TransactionID, err))
			continue
		}

		transaction := &models.BankTransaction{
			TransactionID:   input.TransactionID,
			AccountNumber:   input.AccountNumber,
			Amount:          input.Amount,
			TransactionDate: input.TransactionDate,
			Description:     input.Description,
			ReferenceNumber: input.ReferenceNumber,
		}

		err := s.bankRepo.InsertBankTransaction(tx, transaction)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to insert transaction %s: %v", input.TransactionID, err))
			continue
		}

		result.RecordsCount++
	}

	auditDetails, _ := json.Marshal(map[string]interface{}{
		"total_records": len(transactions),
		"successful":    result.RecordsCount,
		"failed":        len(result.Errors),
	})

	if result.RecordsCount > 0 {
		audit := &models.ReconciliationAudit{
			Action:  models.AuditActionCreated,
			Details: auditDetails,
			UserID:  "system", // Could be replaced with actual user ID if authentication is implemented
		}
		err = s.reconciliationRepo.CreateAuditEntry(tx, audit)
		if err != nil {
			return nil, fmt.Errorf("failed to create audit entry: %v", err)
		}
	}

	// Update result status
	result.Success = len(result.Errors) == 0
	result.Details["total_records"] = len(transactions)
	result.Details["successful"] = result.RecordsCount
	result.Details["failed"] = len(result.Errors)

	if result.Success {
		err = tx.Commit()
		if err != nil {
			return nil, fmt.Errorf("failed to commit transaction: %v", err)
		}
	}

	return result, nil
}

func (s *DataIngestionService) IngestAccountingEntries(entries []AccountingEntryInput) (*IngestionResult, error) {
	result := &IngestionResult{
		Success: true,
		Details: make(map[string]interface{}),
	}

	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	for _, input := range entries {
		// Validate input
		if err := validateAccountingEntry(input); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Invalid entry %s: %v", input.EntryID, err))
			continue
		}

		// Convert to model
		entry := &models.AccountingEntry{
			EntryID:       input.EntryID,
			AccountCode:   input.AccountCode,
			Amount:        input.Amount,
			EntryDate:     input.EntryDate,
			Description:   input.Description,
			InvoiceNumber: input.InvoiceNumber,
		}

		// Insert entry
		err := s.accountingRepo.InsertAccountingEntry(tx, entry)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to insert entry %s: %v", input.EntryID, err))
			continue
		}

		result.RecordsCount++
	}

	// Create audit entry
	auditDetails, _ := json.Marshal(map[string]interface{}{
		"total_records": len(entries),
		"successful":    result.RecordsCount,
		"failed":        len(result.Errors),
	})

	if result.RecordsCount > 0 {
		audit := &models.ReconciliationAudit{
			Action:  models.AuditActionCreated,
			Details: auditDetails,
			UserID:  "system", // Could be replaced with actual user ID if authentication is implemented
		}
		err = s.reconciliationRepo.CreateAuditEntry(tx, audit)
		if err != nil {
			return nil, fmt.Errorf("failed to create audit entry: %v", err)
		}
	}

	// Update result status
	result.Success = len(result.Errors) == 0
	result.Details["total_records"] = len(entries)
	result.Details["successful"] = result.RecordsCount
	result.Details["failed"] = len(result.Errors)

	if result.Success {
		err = tx.Commit()
		if err != nil {
			return nil, fmt.Errorf("failed to commit transaction: %v", err)
		}
	}

	return result, nil
}

func validateBankTransaction(input BankTransactionInput) error {
	if input.TransactionID == "" {
		return fmt.Errorf("transaction_id is required")
	}
	if input.AccountNumber == "" {
		return fmt.Errorf("account_number is required")
	}
	if input.Amount == 0 {
		return fmt.Errorf("amount is required and must be non-zero")
	}
	if input.TransactionDate == "" {
		return fmt.Errorf("transaction_date is required")
	}
	return nil
}

func validateAccountingEntry(input AccountingEntryInput) error {
	if input.EntryID == "" {
		return fmt.Errorf("entry_id is required")
	}
	if input.AccountCode == "" {
		return fmt.Errorf("account_code is required")
	}
	if input.Amount == 0 {
		return fmt.Errorf("amount is required and must be non-zero")
	}
	if input.EntryDate == "" {
		return fmt.Errorf("entry_date is required")
	}
	return nil
}
