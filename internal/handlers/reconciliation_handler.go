package handlers

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"

	"reconciliation-service/internal/models"
	"reconciliation-service/internal/services"
)

type ReconciliationHandler struct {
	reconciliationService *services.ReconciliationService
	processingMutex       sync.Mutex
	activeProcesses       map[string]bool
}

func NewReconciliationHandler(reconciliationService *services.ReconciliationService) *ReconciliationHandler {
	return &ReconciliationHandler{
		reconciliationService: reconciliationService,
		activeProcesses:       make(map[string]bool),
	}
}

func (h *ReconciliationHandler) StartReconciliation(w http.ResponseWriter, r *http.Request) {
	var request struct {
		FromDate string `json:"from_date"`
		ToDate   string `json:"to_date"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	// Validate dates
	if request.FromDate == "" || request.ToDate == "" {
		respondWithError(w, http.StatusBadRequest, "Both from_date and to_date are required")
		return
	}

	_, err := time.Parse("2006-01-02", request.FromDate)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid from_date format. Use YYYY-MM-DD")
		return
	}

	_, err = time.Parse("2006-01-02", request.ToDate)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid to_date format. Use YYYY-MM-DD")
		return
	}

	processKey := request.FromDate + "_" + request.ToDate

	h.processingMutex.Lock()
	if h.activeProcesses[processKey] {
		h.processingMutex.Unlock()
		respondWithError(w, http.StatusConflict, "Reconciliation for this date range is already in progress")
		return
	}
	h.activeProcesses[processKey] = true
	h.processingMutex.Unlock()

	defer func() {
		h.processingMutex.Lock()
		delete(h.activeProcesses, processKey)
		h.processingMutex.Unlock()
	}()

	bankChan := make(chan []*models.BankTransaction, 1)
	accountingChan := make(chan []*models.AccountingEntry, 1)
	errorChan := make(chan error, 2)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		bankTransactions, err := h.reconciliationService.GetBankTransactions(request.FromDate, request.ToDate)
		if err != nil {
			errorChan <- err
			return
		}
		bankChan <- bankTransactions
	}()

	go func() {
		defer wg.Done()
		accountingEntries, err := h.reconciliationService.GetAccountingEntries(request.FromDate, request.ToDate)
		if err != nil {
			errorChan <- err
			return
		}
		accountingChan <- accountingEntries
	}()

	wg.Wait()
	close(bankChan)
	close(accountingChan)
	close(errorChan)

	select {
	case err := <-errorChan:
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}
	default:
		// No errors, continue processing
	}

	var bankTransactions []*models.BankTransaction
	var accountingEntries []*models.AccountingEntry

	select {
	case bankTransactions = <-bankChan:
	default:
		respondWithError(w, http.StatusInternalServerError, "Failed to retrieve bank transactions")
		return
	}

	select {
	case accountingEntries = <-accountingChan:
	default:
		respondWithError(w, http.StatusInternalServerError, "Failed to retrieve accounting entries")
		return
	}

	result, err := h.reconciliationService.ProcessReconciliationWithData(request.FromDate, request.ToDate, bankTransactions, accountingEntries)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, result)
}

func (h *ReconciliationHandler) GetReconciliationStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	batchID := vars["batch_id"]

	if batchID == "" {
		respondWithError(w, http.StatusBadRequest, "Batch ID is required")
		return
	}

	result, err := h.reconciliationService.GetReconciliationStatus(batchID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, result)
}

func (h *ReconciliationHandler) ResolveDispute(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	batchID := vars["batch_id"]

	if batchID == "" {
		respondWithError(w, http.StatusBadRequest, "Batch ID is required")
		return
	}

	var resolution map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&resolution); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	err := h.reconciliationService.ResolveDispute(batchID, resolution)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]string{
		"message":  "Dispute resolved successfully",
		"batch_id": batchID,
	})
}

func (h *ReconciliationHandler) GetUnmatchedRecords(w http.ResponseWriter, r *http.Request) {
	fromDate := r.URL.Query().Get("from_date")
	toDate := r.URL.Query().Get("to_date")

	if fromDate == "" || toDate == "" {
		respondWithError(w, http.StatusBadRequest, "Both from_date and to_date query parameters are required")
		return
	}

	_, err := time.Parse("2006-01-02", fromDate)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid from_date format. Use YYYY-MM-DD")
		return
	}

	_, err = time.Parse("2006-01-02", toDate)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid to_date format. Use YYYY-MM-DD")
		return
	}

	result, err := h.reconciliationService.GetUnmatchedRecords(fromDate, toDate)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, result)
}

func respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, map[string]string{"error": message})
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, err := json.Marshal(payload)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "Error marshaling JSON response"}`))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}
