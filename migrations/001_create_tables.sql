-- Table for Source A: Bank Transactions
CREATE TABLE IF NOT EXISTS bank_transactions (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    transaction_id VARCHAR(100) UNIQUE NOT NULL,
    account_number VARCHAR(50) NOT NULL,
    amount DECIMAL(15,2) NOT NULL,
    transaction_date DATE NOT NULL,
    description TEXT,
    reference_number VARCHAR(100),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_transaction_date (transaction_date),
    INDEX idx_amount (amount),
    INDEX idx_reference (reference_number)
);

-- Table for Source B: Accounting Entries
CREATE TABLE IF NOT EXISTS accounting_entries (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    entry_id VARCHAR(100) UNIQUE NOT NULL,
    account_code VARCHAR(50) NOT NULL,
    amount DECIMAL(15,2) NOT NULL,
    entry_date DATE NOT NULL,
    description TEXT,
    invoice_number VARCHAR(100),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_entry_date (entry_date),
    INDEX idx_amount (amount),
    INDEX idx_invoice (invoice_number)
);

-- Table for Reconciliation Results
CREATE TABLE IF NOT EXISTS reconciliations (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    reconciliation_batch_id VARCHAR(100) NOT NULL,
    status ENUM('matched', 'unmatched_bank', 'unmatched_accounting', 'disputed') NOT NULL,
    match_confidence DECIMAL(3,2),
    amount_difference DECIMAL(15,2) DEFAULT 0.00,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_batch (reconciliation_batch_id),
    INDEX idx_status (status)
);

-- Mapping Table for Relationships
CREATE TABLE IF NOT EXISTS reconciliation_mappings (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    reconciliation_id BIGINT NOT NULL,
    bank_transaction_id BIGINT,
    accounting_entry_id BIGINT,
    mapping_type ENUM('one_to_one', 'one_to_many', 'many_to_one') NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (reconciliation_id) REFERENCES reconciliations(id) ON DELETE CASCADE,
    FOREIGN KEY (bank_transaction_id) REFERENCES bank_transactions(id),
    FOREIGN KEY (accounting_entry_id) REFERENCES accounting_entries(id),
    INDEX idx_reconciliation (reconciliation_id)
);

-- Audit Trail Table
CREATE TABLE IF NOT EXISTS reconciliation_audit (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    reconciliation_id BIGINT NOT NULL,
    action ENUM('created', 'matched', 'unmatched', 'disputed', 'resolved') NOT NULL,
    details JSON,
    user_id VARCHAR(100),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (reconciliation_id) REFERENCES reconciliations(id) ON DELETE CASCADE,
    INDEX idx_reconciliation_audit (reconciliation_id),
    INDEX idx_action (action)
);
