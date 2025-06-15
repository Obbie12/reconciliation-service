-- Drop tables in reverse order to handle foreign key constraints
DROP TABLE IF EXISTS reconciliation_audit;
DROP TABLE IF EXISTS reconciliation_mappings;
DROP TABLE IF EXISTS reconciliations;
DROP TABLE IF EXISTS accounting_entries;
DROP TABLE IF EXISTS bank_transactions;
