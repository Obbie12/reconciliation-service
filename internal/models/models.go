package models

import (
	"database/sql"
	"encoding/json"
	"time"
)

// BankTransaction represents a bank statement transaction
type BankTransaction struct {
	ID              int64     `db:"id" json:"id"`
	TransactionID   string    `db:"transaction_id" json:"transaction_id"`
	AccountNumber   string    `db:"account_number" json:"account_number"`
	Amount          float64   `db:"amount" json:"amount"`
	TransactionDate string    `db:"transaction_date" json:"transaction_date"`
	Description     string    `db:"description" json:"description"`
	ReferenceNumber string    `db:"reference_number" json:"reference_number"`
	CreatedAt       time.Time `db:"created_at" json:"-"`
	UpdatedAt       time.Time `db:"updated_at" json:"-"`
}

// AccountingEntry represents an internal accounting system entry
type AccountingEntry struct {
	ID            int64     `db:"id" json:"id"`
	EntryID       string    `db:"entry_id" json:"entry_id"`
	AccountCode   string    `db:"account_code" json:"account_code"`
	Amount        float64   `db:"amount" json:"amount"`
	EntryDate     string    `db:"entry_date" json:"entry_date"`
	Description   string    `db:"description" json:"description"`
	InvoiceNumber string    `db:"invoice_number" json:"invoice_number"`
	CreatedAt     time.Time `db:"created_at" json:"-"`
	UpdatedAt     time.Time `db:"updated_at" json:"-"`
}

// Reconciliation represents a reconciliation record
type Reconciliation struct {
	ID               int64     `db:"id" json:"id"`
	BatchID          string    `db:"reconciliation_batch_id" json:"reconciliation_batch_id"`
	Status           string    `db:"status" json:"status"`
	MatchConfidence  float64   `db:"match_confidence" json:"match_confidence"`
	AmountDifference float64   `db:"amount_difference" json:"amount_difference"`
	CreatedAt        time.Time `db:"created_at" json:"-"`
	UpdatedAt        time.Time `db:"updated_at" json:"-"`
}

// ReconciliationMapping represents the relationship between transactions and entries
type ReconciliationMapping struct {
	ID                int64         `db:"id" json:"id"`
	ReconciliationID  int64         `db:"reconciliation_id" json:"reconciliation_id"`
	BankTransactionID sql.NullInt64 `db:"bank_transaction_id" json:"bank_transaction_id"`
	AccountingEntryID sql.NullInt64 `db:"accounting_entry_id" json:"accounting_entry_id"`
	MappingType       string        `db:"mapping_type" json:"mapping_type"`
	CreatedAt         time.Time     `db:"created_at" json:"-"`
}

// ReconciliationAudit represents an audit trail entry
type ReconciliationAudit struct {
	ID               int64           `db:"id" json:"id"`
	ReconciliationID int64           `db:"reconciliation_id" json:"reconciliation_id"`
	Action           string          `db:"action" json:"action"`
	Details          json.RawMessage `db:"details" json:"details"`
	UserID           string          `db:"user_id" json:"user_id"`
	CreatedAt        time.Time       `db:"created_at" json:"-"`
}

// ReconciliationStatus constants
const (
	StatusMatched             = "matched"
	StatusUnmatchedBank       = "unmatched_bank"
	StatusUnmatchedAccounting = "unmatched_accounting"
	StatusDisputed            = "disputed"
)

// MappingType constants
const (
	MappingOneToOne  = "one_to_one"
	MappingOneToMany = "one_to_many"
	MappingManyToOne = "many_to_one"
)

// AuditAction constants
const (
	AuditActionCreated   = "created"
	AuditActionMatched   = "matched"
	AuditActionUnmatched = "unmatched"
	AuditActionDisputed  = "disputed"
	AuditActionResolved  = "resolved"
)
