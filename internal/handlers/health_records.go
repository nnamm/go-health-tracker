package handlers

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/nnamm/go-health-tracker/internal/database"
	"github.com/nnamm/go-health-tracker/internal/models"
)

type HealthRecordHandler struct {
	DB database.DBInterface
}

func NewHealthRecordHandler(db database.DBInterface) *HealthRecordHandler {
	return &HealthRecordHandler{DB: db}
}

func (h *HealthRecordHandler) CreateHealthRecord(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading request body: %v", err)
		http.Error(w, "Error reading request body", http.StatusBadRequest)
		return
	}
	log.Printf("Received request body: %s", string(body))

	if len(body) == 0 {
		log.Print("Empty request body received")
		http.Error(w, "Empty request body", http.StatusBadRequest)
		return
	}

	var hr models.HealthRecord
	if err = json.Unmarshal(body, &hr); err != nil {
		log.Printf("Error unmarshaling JSON: %v", err)
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	log.Printf("Parsed HealthRecord: %+v", hr)

	if hr.Date.IsZero() {
		log.Print("Invalid date: zero value")
		http.Error(w, "Invalid date", http.StatusBadRequest)
		return
	}

	if err := h.DB.CreateHealthRecord(&hr); err != nil {
		log.Printf("Error creating health record: %v", err)
		http.Error(w, "Failed to create health record: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(hr)
}

func (h *HealthRecordHandler) GetHealthRecord(w http.ResponseWriter, r *http.Request) {
	dateStr := r.URL.Query().Get("date")
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		http.Error(w, "Invalid date format. Use YYYY-MM-DD", http.StatusBadRequest)
		return
	}

	hr, err := h.DB.ReadHealthRecord(date)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(hr)
}

// GetHealthRecords retrieves record(s) for the specified date (year, month. date)
func (h *HealthRecordHandler) GetHealthRecords(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	var records []models.HealthRecord
	var err error

	switch {
	case query.Get("date") != "":
		date := query.Get("date")
		records, err = h.getByDate(date)
	case query.Get("year") != "":
		year := query.Get("year")
		month := query.Get("month")
		records, err = h.getByYearMonth(year, month)
	default:
		http.Error(w, "Invalid query parameters", http.StatusBadRequest)
		return
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(records)
}

// getByDate retrieves a record for the specified date (YYYYMMDD)
func (h *HealthRecordHandler) getByDate(dateStr string) ([]models.HealthRecord, error) {
	date, err := time.Parse("20060102", dateStr)
	if err != nil {
		return nil, err
	}
	record, err := h.DB.ReadHealthRecord(date)
	if err != nil {
		return nil, err
	}
	if record == nil { // Not exist record
		return []models.HealthRecord{}, nil
	}

	return []models.HealthRecord{*record}, nil
}

// getByYearMonth retrieves record(s) for the specified Year and month (YYYY, MM)
func (h *HealthRecordHandler) getByYearMonth(yearStr, monthStr string) ([]models.HealthRecord, error) {
	year, err := time.Parse("2006", yearStr)
	if err != nil {
		return nil, err
	}

	if monthStr == "" {
		return h.DB.ReadHealthRecordsByYear(year.Year())
	}

	month, err := time.Parse("01", monthStr)
	if err != nil {
		return nil, err
	}

	return h.DB.ReadHealthRecordsByYearMonth(year.Year(), int(month.Month()))
}

func (h *HealthRecordHandler) UpdateHealthRecord(w http.ResponseWriter, r *http.Request) {
	var hr models.HealthRecord
	if err := json.NewDecoder(r.Body).Decode(&hr); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.DB.UpdateHealthRecord(&hr); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(hr)
}

func (h *HealthRecordHandler) DeleteHealthRecord(w http.ResponseWriter, r *http.Request) {
	dateStr := r.URL.Query().Get("date")
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		http.Error(w, "Invalid date format. Use YYYY-MM-DD", http.StatusBadRequest)
		return
	}

	if err = h.DB.DeleteHealthRecord(date); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
