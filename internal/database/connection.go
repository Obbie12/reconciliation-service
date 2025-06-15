package database

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"reconciliation-service/internal/config"

	_ "github.com/go-sql-driver/mysql"
)

func NewConnection(cfg *config.Config) (*sql.DB, error) {
	db, err := sql.Open("mysql", cfg.GetDSN())
	if err != nil {
		return nil, fmt.Errorf("error opening database: %v", err)
	}

	err = db.Ping()
	if err != nil {
		if strings.Contains(err.Error(), "Unknown database") {
			log.Printf("Database '%s' does not exist, attempting to create it...", cfg.Database.Name)

			db.Close()

			rootDSN := getRootDSN(cfg)
			rootDB, err := sql.Open("mysql", rootDSN)
			if err != nil {
				return nil, fmt.Errorf("error connecting to MySQL root: %v", err)
			}
			defer rootDB.Close()
			_, err = rootDB.Exec(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci", cfg.Database.Name))
			if err != nil {
				return nil, fmt.Errorf("error creating database: %v", err)
			}

			log.Printf("Successfully created database '%s'", cfg.Database.Name)

			db, err = sql.Open("mysql", cfg.GetDSN())
			if err != nil {
				return nil, fmt.Errorf("error connecting to new database: %v", err)
			}

			if err = db.Ping(); err != nil {
				return nil, fmt.Errorf("error verifying connection to new database: %v", err)
			}
		} else {
			return nil, fmt.Errorf("error pinging database: %v", err)
		}
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	log.Println("Successfully connected to MySQL database")
	return db, nil
}

func getRootDSN(cfg *config.Config) string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/?parseTime=true",
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Host,
		cfg.Database.Port,
	)
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
