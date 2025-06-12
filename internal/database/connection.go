package database

import (
	"database/sql"
	"log"
	"time"

	"reconciliation-service/internal/config"

	_ "github.com/go-sql-driver/mysql"
)

func NewConnection(cfg *config.Config) (*sql.DB, error) {
	db, err := sql.Open("mysql", cfg.MySQLDSN)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		return nil, err
	}

	log.Println("Successfully connected to MySQL database")
	return db, nil
}

type Transaction struct {
	*sql.Tx
}

func BeginTx(db *sql.DB) (*Transaction, error) {
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	return &Transaction{tx}, nil
}

func (tx *Transaction) Commit() error {
	return tx.Tx.Commit()
}

func (tx *Transaction) Rollback() error {
	return tx.Tx.Rollback()
}
