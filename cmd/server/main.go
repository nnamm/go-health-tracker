package main

import (
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/nnamm/go-health-tracker/internal/database"
	"github.com/nnamm/go-health-tracker/internal/handlers"
)

const (
	healthRecordsPath = "/health/records"
)

func main() {
	// Configure database connection settings
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./health_tracker.db"
	}
	db, err := database.NewDB(dbPath)
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}
	defer db.Close()

	// Initialize handler
	healthHandler := handlers.NewHealthRecordHandler(db)

	// Register route handlers
	http.HandleFunc("/", logMiddleware(routeHandler(healthHandler)))

	// Start the server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}
	log.Printf("Server is running on http://localhost:%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

// routeHandler handles all routes for the API
func routeHandler(handler *handlers.HealthRecordHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Set common response headers
		setCommonHeaders(w)

		// Route based on path
		switch strings.TrimSuffix(r.URL.Path, "/") {
		case healthRecordsPath:
			handleHealthRecords(handler, w, r)
		default:
			http.NotFound(w, r)
		}
	}
}

// handleHealthRecords handles /health/reorords endpoints
func handleHealthRecords(handler *handlers.HealthRecordHandler, w http.ResponseWriter, r *http.Request) {
	// CORS preflight request support
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	switch r.Method {
	case http.MethodGet:
		handler.GetHealthRecords(w, r)
	case http.MethodPost:
		handler.CreateHealthRecord(w, r)
	case http.MethodPut:
		handler.UpdateHealthRecord(w, r)
	case http.MethodDelete:
		handler.DeleteHealthRecord(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// setCommonHeaders sets common response headers
func setCommonHeaders(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*") // CORS
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
}

// logMiddleware logs request details
func logMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()

		// call the warapped handler
		next(w, r)

		// Log the request details
		log.Printf(
			"[%s] %s %s %s",
			r.Method,
			r.URL.Path,
			r.RemoteAddr,
			time.Since(startTime),
		)
	}
}
