// Helath Tracker API is RESTful API for tracking health-record data.
// Currently supports step count recording, with plans to add other health matrics in the future.
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

// API path constants
const (
	healthRecordsPath = "/health/records"
)

// main is the application entry point.
// It initializes the database connection, configures routing, and starts the HTTP server.
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

// routeHandler returns a handler function that processes all API routes.
// This handler forwards incoming HTTP requests to the appropriate endpoint handler.
// It also handles response header configuration and path normalization.
//
// Currently supported endpoints:
// - /health/records - Health record management (GET, POST, PUT, DELETE)
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

// handleHealthMethod processes HTTP methods (GET, POST, PUT, DELETE) for health records.
// It calls the appropriate handler function based on the method:
// - GET: Retrieve health records
// - POST: Create a new health record
// - PUT: Update an existing health record
// - DELETE: Delete a health record
//
// It also handles CORS preflight requests (OPTIONS).
// Unsupported HTTP methods receive a 405 Method Not Allowed response.
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

// setCommonHeaders sets common HTTP headers for all responses.
// Headers set:
// - Content-Type: application/json
// - Access-Control-Allow-Origin: * (CORS support)
// - Access-Control-Allow-Methods
// - Access-Control-Allow-Headers
//
// Note: More restrictive CORS settings are recommended for production environments.
func setCommonHeaders(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*") // CORS
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
}

// logMiddleware is middleware that logs HTTP request details.
// It records the request method, path, client IP address, and processing time
// to the log output.
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
