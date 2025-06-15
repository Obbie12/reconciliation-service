package repositories

import (
	"database/sql"
	"errors"
	"time"

	"reconciliation-service/internal/models"
)

type ReconciliationRepository interface {
	CreateReconciliation(tx *sql.Tx, rec *models.Reconciliation) error
	GetReconciliationByID(id int64) (*models.Reconciliation, error)
	GetReconciliationByBatchID(batchID string) (*models.Reconciliation, error)
	UpdateReconciliationStatus(tx *sql.Tx, id int64, status string) error
	CreateMapping(tx *sql.Tx, mapping *models.ReconciliationMapping) error
	CreateAuditEntry(tx *sql.Tx, audit *models.ReconciliationAudit) error
	GetUnmatchedRecords(fromDate, toDate string) (map[string]interface{}, error)
}

type reconciliationRepository struct {
	db *sql.DB
}

func NewReconciliationRepository(db *sql.DB) ReconciliationRepository {
	return &reconciliationRepository{db: db}
}

func (r *reconciliationRepository) CreateReconciliation(tx *sql.Tx, rec *models.Reconciliation) error {
	query := `
		INSERT INTO reconciliations (
			reconciliation_batch_id, status, match_confidence, amount_difference
		) VALUES (?, ?, ?, ?)
	`
	result, err := tx.Exec(query,
		rec.BatchID,
		rec.Status,
		rec.MatchConfidence,
		rec.AmountDifference,
	)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	rec.ID = id
	return nil
}

func (r *reconciliationRepository) GetReconciliationByID(id int64) (*models.Reconciliation, error) {
	rec := &models.Reconciliation{}
	query := `
		SELECT id, reconciliation_batch_id, status, match_confidence,
		       amount_difference, created_at, updated_at
		FROM reconciliations
		WHERE id = ?
	`
	err := r.db.QueryRow(query, id).Scan(
		&rec.ID,
		&rec.BatchID,
		&rec.Status,
		&rec.MatchConfidence,
		&rec.AmountDifference,
		&rec.CreatedAt,
		&rec.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, errors.New("reconciliation not found")
	}
	if err != nil {
		return nil, err
	}
	return rec, nil
}

func (r *reconciliationRepository) GetReconciliationByBatchID(batchID string) (*models.Reconciliation, error) {
	rec := &models.Reconciliation{}
	query := `
		SELECT id, reconciliation_batch_id, status, match_confidence,
		       amount_difference, created_at, updated_at
		FROM reconciliations
		WHERE reconciliation_batch_id = ?
	`
	err := r.db.QueryRow(query, batchID).Scan(
		&rec.ID,
		&rec.BatchID,
		&rec.Status,
		&rec.MatchConfidence,
		&rec.AmountDifference,
		&rec.CreatedAt,
		&rec.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, errors.New("reconciliation not found")
	}
	if err != nil {
		return nil, err
	}
	return rec, nil
}

func (r *reconciliationRepository) UpdateReconciliationStatus(tx *sql.Tx, id int64, status string) error {
	query := `
		UPDATE reconciliations
		SET status = ?,
		    updated_at = ?
		WHERE id = ?
	`
	result, err := tx.Exec(query, status, time.Now(), id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return errors.New("reconciliation not found")
	}
	return nil
}

func (r *reconciliationRepository) CreateMapping(tx *sql.Tx, mapping *models.ReconciliationMapping) error {
	query := `
		INSERT INTO reconciliation_mappings (
			reconciliation_id, bank_transaction_id, accounting_entry_id, mapping_type
		) VALUES (?, ?, ?, ?)
	`
	result, err := tx.Exec(query,
		mapping.ReconciliationID,
		mapping.BankTransactionID,
		mapping.AccountingEntryID,
		mapping.MappingType,
	)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	mapping.ID = id
	return nil
}

func (r *reconciliationRepository) CreateAuditEntry(tx *sql.Tx, audit *models.ReconciliationAudit) error {
	query := `
		INSERT INTO reconciliation_audit (
			reconciliation_id, action, details, user_id
		) VALUES (?, ?, ?, ?)
	`
	result, err := tx.Exec(query,
		audit.ReconciliationID,
		audit.Action,
		audit.Details,
		audit.UserID,
	)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	audit.ID = id
	return nil
}

func (r *reconciliationRepository) GetUnmatchedRecords(fromDate, toDate string) (map[string]interface{}, error) {
	bankQuery := `
		SELECT bt.id, bt.transaction_id, bt.amount, bt.transaction_date
		FROM bank_transactions bt
		LEFT JOIN reconciliation_mappings rm ON bt.id = rm.bank_transaction_id
		WHERE rm.id IS NULL
		AND bt.transaction_date BETWEEN ? AND ?
	`
	bankRows, err := r.db.Query(bankQuery, fromDate, toDate)
	if err != nil {
		return nil, err
	}
	defer bankRows.Close()

	var unmatchedBankTransactions []map[string]interface{}
	for bankRows.Next() {
		var id int64
		var transactionID string
		var amount float64
		var transactionDate string

		err := bankRows.Scan(&id, &transactionID, &amount, &transactionDate)
		if err != nil {
			return nil, err
		}

		unmatchedBankTransactions = append(unmatchedBankTransactions, map[string]interface{}{
			"id":               id,
			"transaction_id":   transactionID,
			"amount":           amount,
			"transaction_date": transactionDate,
		})
	}

	// Get unmatched accounting entries
	accountingQuery := `
		SELECT ae.id, ae.entry_id, ae.amount, ae.entry_date
		FROM accounting_entries ae
		LEFT JOIN reconciliation_mappings rm ON ae.id = rm.accounting_entry_id
		WHERE rm.id IS NULL
		AND ae.entry_date BETWEEN ? AND ?
	`
	accountingRows, err := r.db.Query(accountingQuery, fromDate, toDate)
	if err != nil {
		return nil, err
	}
	defer accountingRows.Close()

	var unmatchedAccountingEntries []map[string]interface{}
	for accountingRows.Next() {
		var id int64
		var entryID string
		var amount float64
		var entryDate string

		err := accountingRows.Scan(&id, &entryID, &amount, &entryDate)
		if err != nil {
			return nil, err
		}

		unmatchedAccountingEntries = append(unmatchedAccountingEntries, map[string]interface{}{
			"id":         id,
			"entry_id":   entryID,
			"amount":     amount,
			"entry_date": entryDate,
		})
	}

	return map[string]interface{}{
		"unmatched_bank_transactions":  unmatchedBankTransactions,
		"unmatched_accounting_entries": unmatchedAccountingEntries,
	}, nil
}
