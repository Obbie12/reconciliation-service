package handlers

import (
	"encoding/json"
	"net/http"

	"reconciliation-service/internal/services"
)

type DataHandler struct {
	dataIngestionService *services.DataIngestionService
}

func NewDataHandler(dataIngestionService *services.DataIngestionService) *DataHandler {
	return &DataHandler{
		dataIngestionService: dataIngestionService,
	}
}

func (h *DataHandler) IngestBankTransactions(w http.ResponseWriter, r *http.Request) {
	var transactions []services.BankTransactionInput

	// Parse request body
	if err := json.NewDecoder(r.Body).Decode(&transactions); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	// Validate request
	if len(transactions) == 0 {
		respondWithError(w, http.StatusBadRequest, "No transactions provided")
		return
	}

	// Process transactions
	result, err := h.dataIngestionService.IngestBankTransactions(transactions)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Return response
	status := http.StatusOK
	if !result.Success {
		status = http.StatusPartialContent
	}
	respondWithJSON(w, status, result)
}

func (h *DataHandler) IngestAccountingEntries(w http.ResponseWriter, r *http.Request) {
	var entries []services.AccountingEntryInput

	// Parse request body
	if err := json.NewDecoder(r.Body).Decode(&entries); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	// Validate request
	if len(entries) == 0 {
		respondWithError(w, http.StatusBadRequest, "No entries provided")
		return
	}

	// Process entries
	result, err := h.dataIngestionService.IngestAccountingEntries(entries)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Return response
	status := http.StatusOK
	if !result.Success {
		status = http.StatusPartialContent
	}
	respondWithJSON(w, status, result)
}

type BankTransactionsRequest struct {
	Transactions []services.BankTransactionInput `json:"transactions"`
}

type AccountingEntriesRequest struct {
	Entries []services.AccountingEntryInput `json:"entries"`
}
