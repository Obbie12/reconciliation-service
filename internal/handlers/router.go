package handlers

import (
	"database/sql"
	"net/http"

	"github.com/gorilla/mux"

	"reconciliation-service/internal/config"
	"reconciliation-service/internal/repositories"
	"reconciliation-service/internal/services"
)

func SetupRouter(db *sql.DB, cfg *config.Config) *mux.Router {
	router := mux.NewRouter()

	// Initialize repositories
	bankRepo := repositories.NewBankRepository(db)
	accountingRepo := repositories.NewAccountingRepository(db)
	reconciliationRepo := repositories.NewReconciliationRepository(db)

	// Initialize services
	reconciliationService := services.NewReconciliationService(
		db,
		bankRepo,
		accountingRepo,
		reconciliationRepo,
	)

	dataIngestionService := services.NewDataIngestionService(
		db,
		bankRepo,
		accountingRepo,
		reconciliationRepo,
	)

	// Initialize handlers
	reconciliationHandler := NewReconciliationHandler(reconciliationService)
	dataHandler := NewDataHandler(dataIngestionService)

	// API versioning
	api := router.PathPrefix("/api/v1").Subrouter()

	// Middleware
	api.Use(loggingMiddleware)
	api.Use(jsonContentTypeMiddleware)

	// Reconciliation endpoints
	api.HandleFunc("/reconciliation/start", reconciliationHandler.StartReconciliation).Methods(http.MethodPost)
	api.HandleFunc("/reconciliation/{batch_id}/status", reconciliationHandler.GetReconciliationStatus).Methods(http.MethodGet)
	api.HandleFunc("/reconciliation/{batch_id}/resolve", reconciliationHandler.ResolveDispute).Methods(http.MethodPost)
	api.HandleFunc("/reconciliation/unmatched", reconciliationHandler.GetUnmatchedRecords).Methods(http.MethodGet)

	api.HandleFunc("/data/bank-transactions", dataHandler.IngestBankTransactions).Methods(http.MethodPost)
	api.HandleFunc("/data/accounting-entries", dataHandler.IngestAccountingEntries).Methods(http.MethodPost)

	// Health check endpoint
	router.HandleFunc("/health", healthCheckHandler).Methods(http.MethodGet)

	return router
}

// Middleware functions

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Log the request
		// log.Printf("%s %s %s", r.RemoteAddr, r.Method, r.URL)
		next.ServeHTTP(w, r)
	})
}

func jsonContentTypeMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	})
}

func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	response := map[string]string{
		"status": "healthy",
	}
	respondWithJSON(w, http.StatusOK, response)
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type SuccessResponse struct {
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}
