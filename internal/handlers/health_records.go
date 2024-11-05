package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/nnamm/go-health-tracker/internal/apperr"
	"github.com/nnamm/go-health-tracker/internal/database"
	"github.com/nnamm/go-health-tracker/internal/models"
)

// HealthRecordHandler handles HTTP requests for health records
type HealthRecordHandler struct {
	DB database.DBInterface
}

// HealthRecordResult represents the response structure for health records
type HealthRecordResult struct {
	Records []models.HealthRecord `json:"records"`
}

// NewHealthRecordHandler creates a new NewHealthRecordHandler
func NewHealthRecordHandler(db database.DBInterface) *HealthRecordHandler {
	return &HealthRecordHandler{DB: db}
}

// CreateHealthRecord handles the creation of a new health record
func (h *HealthRecordHandler) CreateHealthRecord(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.handleError(w, apperr.NewAppError(apperr.ErrorTypeInternalServer, "failed to read request body: "+err.Error()))
		return
	}

	var hr models.HealthRecord
	if err = hr.UnmarshalJSON(body); err != nil {
		h.handleError(w, apperr.NewAppError(apperr.ErrorTypeInvalidFormat, err.Error()))
		return
	}

	if hr.Date.IsZero() {
		h.handleError(w, apperr.NewAppError(apperr.ErrorTypeInvalidDate, "date is required"))
		return
	}

	createdRecord, err := h.DB.CreateHealthRecord(&hr)
	if err != nil {
		h.handleError(w, apperr.NewAppError(apperr.ErrorTypeInternalServer, "failed to create health record: "+err.Error()))
		return
	}

	result := HealthRecordResult{
		Records: []models.HealthRecord{*createdRecord},
	}
	h.sendJSONResponse(w, result, http.StatusCreated)
}

// GetHealthRecords retrieves record(s) for the specified date (year, month. date)
func (h *HealthRecordHandler) GetHealthRecords(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	var result HealthRecordResult
	var err error

	switch {
	case query.Get("date") != "":
		var record *models.HealthRecord
		record, err = h.getByDate(query.Get("date"))
		if record != nil {
			result.Records = []models.HealthRecord{*record}
		}
	case query.Get("year") != "":
		result.Records, err = h.getByYearMonth(query.Get("year"), query.Get("month"))
	default:
		h.sendErrorResponse(w, apperr.NewAppError(apperr.ErrorTypeInvalidFormat, "Invalid query parameters: expected date or year"), http.StatusBadRequest)
		return
	}

	if err != nil {
		h.handleError(w, err)
		return
	}

	h.sendJSONResponse(w, result, http.StatusOK)
}

// UpdateHealthRecord handles the update of an existing health record
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

// DeleteHealthRecord handles the deletion of a health record
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

// getByDate retrieves a record for the specified date (YYYYMMDD)
func (h *HealthRecordHandler) getByDate(dateStr string) (*models.HealthRecord, error) {
	date, err := time.Parse("20060102", dateStr)
	if err != nil {
		return nil, apperr.NewAppError(apperr.ErrorTypeInvalidDate, "Invalid date format: "+dateStr+" (Use YYYYMMDD)")
	}

	record, err := h.DB.ReadHealthRecord(date)
	if err != nil {
		return nil, apperr.NewAppError(apperr.ErrorTypeInternalServer, "Failed to read health record: "+err.Error())
	}
	if record == nil {
		return nil, apperr.NewAppError(apperr.ErrorTypeNotFound, "Unexpected: Health record not found for date: "+dateStr)
	}

	return record, nil
}

// getByYearMonth retrieves record(s) for the specified year and month (YYYY, MM)
func (h *HealthRecordHandler) getByYearMonth(yearStr, monthStr string) ([]models.HealthRecord, error) {
	year, err := time.Parse("2006", yearStr)
	if err != nil {
		return nil, apperr.NewAppError(apperr.ErrorTypeInvalidYear, "Invalid year format: "+yearStr+" (Use YYYY)")
	}

	if monthStr == "" {
		records, err := h.DB.ReadHealthRecordsByYear(year.Year())
		if err != nil {
			return nil, apperr.NewAppError(apperr.ErrorTypeInternalServer, "Failed to read health records: "+err.Error())
		}
		return records, nil
	}

	month, err := time.Parse("01", monthStr)
	if err != nil {
		return nil, apperr.NewAppError(apperr.ErrorTypeInvalidMonth, "Invalid month format: "+monthStr+" (Use MM)")
	}
	records, err := h.DB.ReadHealthRecordsByYearMonth(year.Year(), int(month.Month()))
	if err != nil {
		return nil, apperr.NewAppError(apperr.ErrorTypeInternalServer, "Failed to read  health records: "+err.Error())
	}

	return records, nil
}

// handleError processes errors and sends appropriate responses
func (h *HealthRecordHandler) handleError(w http.ResponseWriter, err error) {
	var appErr apperr.AppError
	if errors.As(err, &appErr) {
		switch appErr.Type {
		case apperr.ErrorTypeInvalidDate, apperr.ErrorTypeInvalidYear, apperr.ErrorTypeInvalidMonth, apperr.ErrorTypeInvalidFormat:
			h.sendErrorResponse(w, appErr, http.StatusBadRequest)
		case apperr.ErrorTypeNotFound:
			h.sendErrorResponse(w, appErr, http.StatusNotFound)
		default:
			h.sendErrorResponse(w, appErr, http.StatusInternalServerError)
		}
	} else {
		h.sendErrorResponse(w, apperr.NewAppError(apperr.ErrorTypeInternalServer, err.Error()), http.StatusInternalServerError)
	}
}

// sendJSONResponse sends a JSON response
func (h *HealthRecordHandler) sendJSONResponse(w http.ResponseWriter, data HealthRecordResult, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.sendErrorResponse(w, apperr.NewAppError(apperr.ErrorTypeInternalServer, "failed to encode response"), http.StatusInternalServerError)
	}
}

// sendErrorResponse sends an error response
func (h *HealthRecordHandler) sendErrorResponse(w http.ResponseWriter, err apperr.AppError, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
