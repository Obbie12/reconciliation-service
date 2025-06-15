package repositories

import (
	"database/sql"
	"errors"
	"time"

	"reconciliation-service/internal/models"
)

type AccountingRepository interface {
	InsertAccountingEntry(tx *sql.Tx, ae *models.AccountingEntry) error
	GetAccountingEntryByID(id int64) (*models.AccountingEntry, error)
	GetAccountingEntryByEntryID(entryID string) (*models.AccountingEntry, error)
	GetUnreconciledEntries(fromDate, toDate string) ([]*models.AccountingEntry, error)
	GetEntriesByAmount(amount float64, fromDate, toDate string) ([]*models.AccountingEntry, error)
	UpdateAccountingEntry(tx *sql.Tx, ae *models.AccountingEntry) error
}

type accountingRepository struct {
	db *sql.DB
}

func NewAccountingRepository(db *sql.DB) AccountingRepository {
	return &accountingRepository{db: db}
}

func (r *accountingRepository) InsertAccountingEntry(tx *sql.Tx, ae *models.AccountingEntry) error {
	query := `
		INSERT INTO accounting_entries (
			entry_id, account_code, amount,
			entry_date, description, invoice_number
		) VALUES (?, ?, ?, ?, ?, ?)
	`
	result, err := tx.Exec(query,
		ae.EntryID,
		ae.AccountCode,
		ae.Amount,
		ae.EntryDate,
		ae.Description,
		ae.InvoiceNumber,
	)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	ae.ID = id
	return nil
}

func (r *accountingRepository) GetAccountingEntryByID(id int64) (*models.AccountingEntry, error) {
	ae := &models.AccountingEntry{}
	query := `
		SELECT id, entry_id, account_code, amount,
		       entry_date, description, invoice_number,
		       created_at, updated_at
		FROM accounting_entries
		WHERE id = ?
	`
	err := r.db.QueryRow(query, id).Scan(
		&ae.ID,
		&ae.EntryID,
		&ae.AccountCode,
		&ae.Amount,
		&ae.EntryDate,
		&ae.Description,
		&ae.InvoiceNumber,
		&ae.CreatedAt,
		&ae.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, errors.New("accounting entry not found")
	}
	if err != nil {
		return nil, err
	}
	return ae, nil
}

func (r *accountingRepository) GetAccountingEntryByEntryID(entryID string) (*models.AccountingEntry, error) {
	ae := &models.AccountingEntry{}
	query := `
		SELECT id, entry_id, account_code, amount,
		       entry_date, description, invoice_number,
		       created_at, updated_at
		FROM accounting_entries
		WHERE entry_id = ?
	`
	err := r.db.QueryRow(query, entryID).Scan(
		&ae.ID,
		&ae.EntryID,
		&ae.AccountCode,
		&ae.Amount,
		&ae.EntryDate,
		&ae.Description,
		&ae.InvoiceNumber,
		&ae.CreatedAt,
		&ae.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, errors.New("accounting entry not found")
	}
	if err != nil {
		return nil, err
	}
	return ae, nil
}

func (r *accountingRepository) GetUnreconciledEntries(fromDate, toDate string) ([]*models.AccountingEntry, error) {
	query := `
		SELECT ae.id, ae.entry_id, ae.account_code, ae.amount,
		       ae.entry_date, ae.description, ae.invoice_number,
		       ae.created_at, ae.updated_at
		FROM accounting_entries ae
		LEFT JOIN reconciliation_mappings rm ON ae.id = rm.accounting_entry_id
		WHERE rm.id IS NULL
		AND ae.entry_date BETWEEN ? AND ?
	`
	rows, err := r.db.Query(query, fromDate, toDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []*models.AccountingEntry
	for rows.Next() {
		ae := &models.AccountingEntry{}
		err := rows.Scan(
			&ae.ID,
			&ae.EntryID,
			&ae.AccountCode,
			&ae.Amount,
			&ae.EntryDate,
			&ae.Description,
			&ae.InvoiceNumber,
			&ae.CreatedAt,
			&ae.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		entries = append(entries, ae)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return entries, nil
}

func (r *accountingRepository) GetEntriesByAmount(amount float64, fromDate, toDate string) ([]*models.AccountingEntry, error) {
	query := `
		SELECT id, entry_id, account_code, amount,
		       entry_date, description, invoice_number,
		       created_at, updated_at
		FROM accounting_entries
		WHERE amount = ?
		AND entry_date BETWEEN ? AND ?
	`
	rows, err := r.db.Query(query, amount, fromDate, toDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []*models.AccountingEntry
	for rows.Next() {
		ae := &models.AccountingEntry{}
		err := rows.Scan(
			&ae.ID,
			&ae.EntryID,
			&ae.AccountCode,
			&ae.Amount,
			&ae.EntryDate,
			&ae.Description,
			&ae.InvoiceNumber,
			&ae.CreatedAt,
			&ae.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		entries = append(entries, ae)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return entries, nil
}

func (r *accountingRepository) UpdateAccountingEntry(tx *sql.Tx, ae *models.AccountingEntry) error {
	query := `
		UPDATE accounting_entries
		SET account_code = ?,
			amount = ?,
			entry_date = ?,
			description = ?,
			invoice_number = ?,
			updated_at = ?
		WHERE id = ?
	`
	result, err := tx.Exec(query,
		ae.AccountCode,
		ae.Amount,
		ae.EntryDate,
		ae.Description,
		ae.InvoiceNumber,
		time.Now(),
		ae.ID,
	)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return errors.New("accounting entry not found")
	}
	return nil
}
