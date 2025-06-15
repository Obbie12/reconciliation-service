package repositories

import (
	"database/sql"
	"errors"
	"time"

	"reconciliation-service/internal/models"
)

type BankRepository interface {
	InsertBankTransaction(tx *sql.Tx, bt *models.BankTransaction) error
	GetBankTransactionByID(id int64) (*models.BankTransaction, error)
	GetBankTransactionByTransactionID(transactionID string) (*models.BankTransaction, error)
	GetUnreconciledTransactions(fromDate, toDate string) ([]*models.BankTransaction, error)
	UpdateBankTransaction(tx *sql.Tx, bt *models.BankTransaction) error
}

type bankRepository struct {
	db *sql.DB
}

func NewBankRepository(db *sql.DB) BankRepository {
	return &bankRepository{db: db}
}

func (r *bankRepository) InsertBankTransaction(tx *sql.Tx, bt *models.BankTransaction) error {
	query := `
		INSERT INTO bank_transactions (
			transaction_id, account_number, amount, 
			transaction_date, description, reference_number
		) VALUES (?, ?, ?, ?, ?, ?)
	`
	result, err := tx.Exec(query,
		bt.TransactionID,
		bt.AccountNumber,
		bt.Amount,
		bt.TransactionDate,
		bt.Description,
		bt.ReferenceNumber,
	)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	bt.ID = id
	return nil
}

func (r *bankRepository) GetBankTransactionByID(id int64) (*models.BankTransaction, error) {
	bt := &models.BankTransaction{}
	query := `
		SELECT id, transaction_id, account_number, amount, 
		       transaction_date, description, reference_number,
		       created_at, updated_at
		FROM bank_transactions
		WHERE id = ?
	`
	err := r.db.QueryRow(query, id).Scan(
		&bt.ID,
		&bt.TransactionID,
		&bt.AccountNumber,
		&bt.Amount,
		&bt.TransactionDate,
		&bt.Description,
		&bt.ReferenceNumber,
		&bt.CreatedAt,
		&bt.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, errors.New("bank transaction not found")
	}
	if err != nil {
		return nil, err
	}
	return bt, nil
}

func (r *bankRepository) GetBankTransactionByTransactionID(transactionID string) (*models.BankTransaction, error) {
	bt := &models.BankTransaction{}
	query := `
		SELECT id, transaction_id, account_number, amount, 
		       transaction_date, description, reference_number,
		       created_at, updated_at
		FROM bank_transactions
		WHERE transaction_id = ?
	`
	err := r.db.QueryRow(query, transactionID).Scan(
		&bt.ID,
		&bt.TransactionID,
		&bt.AccountNumber,
		&bt.Amount,
		&bt.TransactionDate,
		&bt.Description,
		&bt.ReferenceNumber,
		&bt.CreatedAt,
		&bt.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, errors.New("bank transaction not found")
	}
	if err != nil {
		return nil, err
	}
	return bt, nil
}

func (r *bankRepository) GetUnreconciledTransactions(fromDate, toDate string) ([]*models.BankTransaction, error) {
	query := `
		SELECT bt.id, bt.transaction_id, bt.account_number, bt.amount, 
		       bt.transaction_date, bt.description, bt.reference_number,
		       bt.created_at, bt.updated_at
		FROM bank_transactions bt
		LEFT JOIN reconciliation_mappings rm ON bt.id = rm.bank_transaction_id
		WHERE rm.id IS NULL
		AND bt.transaction_date BETWEEN ? AND ?
	`
	rows, err := r.db.Query(query, fromDate, toDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var transactions []*models.BankTransaction
	for rows.Next() {
		bt := &models.BankTransaction{}
		err := rows.Scan(
			&bt.ID,
			&bt.TransactionID,
			&bt.AccountNumber,
			&bt.Amount,
			&bt.TransactionDate,
			&bt.Description,
			&bt.ReferenceNumber,
			&bt.CreatedAt,
			&bt.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		transactions = append(transactions, bt)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return transactions, nil
}

func (r *bankRepository) UpdateBankTransaction(tx *sql.Tx, bt *models.BankTransaction) error {
	query := `
		UPDATE bank_transactions
		SET account_number = ?,
			amount = ?,
			transaction_date = ?,
			description = ?,
			reference_number = ?,
			updated_at = ?
		WHERE id = ?
	`
	result, err := tx.Exec(query,
		bt.AccountNumber,
		bt.Amount,
		bt.TransactionDate,
		bt.Description,
		bt.ReferenceNumber,
		time.Now(),
		bt.ID,
	)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return errors.New("bank transaction not found")
	}
	return nil
}
