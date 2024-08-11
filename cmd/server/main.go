package main

import (
	"log"
	"net/http"
	"os"

	"github.com/nnamm/go-health-tracker/internal/database"
	"github.com/nnamm/go-health-tracker/internal/handlers"
)

func main() {
	// Configure database connection settings
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./health_tracker.db"
	}
	db, err := database.NewDB(dbPath)
	if err != nil {
		log.Fatalf("Failed to connect database: %v", err)
	}
	defer db.Close()

	// Create table
	err = db.CreateTable()
	if err != nil {
		log.Fatalf("Failed to create table: %v", err)
	}

	// Initialize handler
	healthHandler := handlers.NewHealthRecordHandler(db)

	// Configure routing
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			healthHandler.GetHealthRecord(w, r)
		case http.MethodPost:
			healthHandler.CreateHealthRecord(w, r)
		case http.MethodPut:
			healthHandler.UpdateHealthRecord(w, r)
		case http.MethodDelete:
			healthHandler.DeleteHealthRecord(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Start the server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}
	log.Printf("Server is running on http://localhost:%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
