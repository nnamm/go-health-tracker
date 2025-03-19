package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/nnamm/go-health-tracker/internal/apperr"
	"github.com/nnamm/go-health-tracker/internal/config"
	"github.com/nnamm/go-health-tracker/internal/database"
	"github.com/nnamm/go-health-tracker/internal/models"
	"github.com/nnamm/go-health-tracker/internal/validators"
)

// HealthRecordHandler handles HTTP requests for health records
type HealthRecordHandler struct {
	DB        database.DBInterface
	validator validators.HealthRecordValidator
}

// NewHealthRecordHandler creates a new NewHealthRecordHandler
func NewHealthRecordHandler(db database.DBInterface) *HealthRecordHandler {
	return &HealthRecordHandler{
		DB:        db,
		validator: validators.NewHealthRecordValidator(),
	}
}

// HealthRecordResult represents the response structure for health records
type HealthRecordResult struct {
	Records []models.HealthRecord `json:"records"`
}

// CreateHealthRecord handles the creation of a new health record
func (h *HealthRecordHandler) CreateHealthRecord(w http.ResponseWriter, r *http.Request) {
	// set a timeout for the request context
	ctx, cancel := context.WithTimeout(r.Context(), time.Duration(config.RequestTimeoutSecond)*time.Second)
	defer cancel()

	// Create a new request with original request's context
	r = r.WithContext(ctx)

	// Limit the request body size to 8KB
	r.Body = http.MaxBytesReader(w, r.Body, 8*1024)

	// Create channels to handle the request body and errors for async processing
	bodyCh := make(chan []byte, 1)
	errCh := make(chan error, 1)

	go func() {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			errCh <- err
			return
		}
		bodyCh <- body
	}()

	// Check if the request has been cancelled or timed out
	select {
	case <-ctx.Done():
		if ctx.Err() == context.DeadlineExceeded {
			h.handleError(w, apperr.NewAppError(apperr.ErrorTypeInternalServer, "request processing timed out"))
		} else {
			h.handleError(w, apperr.NewAppError(apperr.ErrorTypeInternalServer, "request was cancelled"))
		}
		return
	case err := <-errCh:
		if err.Error() == "http: request body too large" {
			h.handleError(w, apperr.NewAppError(apperr.ErrorTypeBadRequest, "request body too large"))
		} else {
			h.handleError(w, apperr.NewAppError(apperr.ErrorTypeInternalServer, "failed to read request body"))
		}
		return
	case body := <-bodyCh:
		var hr models.HealthRecord
		if err := hr.UnmarshalJSON(body); err != nil {
			h.handleError(w, apperr.NewAppError(apperr.ErrorTypeInvalidFormat, err.Error()))
			return
		}

		if err := h.validator.Validate(&hr); err != nil {
			h.handleError(w, err)
			return
		}

		// Send success response
		createdRecord, err := h.DB.CreateHealthRecord(ctx, &hr)
		if err != nil {
			h.handleError(w, apperr.NewAppError(apperr.ErrorTypeInternalServer, "failed to create health record: "+err.Error()))
			return
		}

		result := HealthRecordResult{
			Records: []models.HealthRecord{*createdRecord},
		}
		h.sendJSONResponse(w, result, http.StatusCreated)
	}
}

// GetHealthRecords retrieves record(s) for the specified date (year, month. date)
func (h *HealthRecordHandler) GetHealthRecords(w http.ResponseWriter, r *http.Request) {
	// set a timeout for the request context
	ctx, cancel := context.WithTimeout(r.Context(), time.Duration(config.RequestTimeoutSecond)*time.Second)
	defer cancel()

	query := r.URL.Query()
	var result HealthRecordResult
	var err error

	switch {
	case query.Get("date") != "":
		var record *models.HealthRecord
		record, err = h.getByDate(ctx, query.Get("date"))
		if record != nil {
			result.Records = []models.HealthRecord{*record}
		}
	case query.Get("year") != "":
		result.Records, err = h.getByYearMonth(ctx, query.Get("year"), query.Get("month"))
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
	// Set a timeout for the request context
	ctx, _ := context.WithTimeout(r.Context(), time.Duration(config.RequestTimeoutSecond)*time.Second)

	// Create a new request with original request's context
	r = r.WithContext(ctx)

	// Limit the request body size to 8KB
	r.Body = http.MaxBytesReader(w, r.Body, 8*1024)

	// Create channels to handle the request body and errors for async processing
	bodyCh := make(chan []byte, 1)
	errCh := make(chan error, 1)

	go func() {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			errCh <- err
			return
		}
		bodyCh <- body
	}()

	// Check if the request has been cancelled or timed out
	select {
	case <-ctx.Done():
		if ctx.Err() == context.DeadlineExceeded {
			h.handleError(w, apperr.NewAppError(apperr.ErrorTypeInternalServer, "request processing timed out"))
		} else {
			h.handleError(w, apperr.NewAppError(apperr.ErrorTypeInternalServer, "request was cancelled"))
		}
	case err := <-errCh:
		if err.Error() == "http: request body too large" {
			h.handleError(w, apperr.NewAppError(apperr.ErrorTypeBadRequest, "request body too large"))
		} else {
			h.handleError(w, apperr.NewAppError(apperr.ErrorTypeInternalServer, "failed to read request body"))
		}
	case body := <-bodyCh:
		var hr models.HealthRecord
		if err := hr.UnmarshalJSON(body); err != nil {
			h.handleError(w, apperr.NewAppError(apperr.ErrorTypeInvalidFormat, err.Error()))
			return
		}

		if err := h.validator.Validate(&hr); err != nil {
			h.handleError(w, err)
			return
		}

		if err := h.DB.UpdateHealthRecord(ctx, &hr); err != nil {
			h.handleError(w, apperr.NewAppError(apperr.ErrorTypeInternalServer, "failed to update health record: "+err.Error()))
			return
		}

		// Send success response
		updatedRecord, err := h.DB.ReadHealthRecord(ctx, hr.Date)
		if err != nil {
			h.handleError(w, apperr.NewAppError(apperr.ErrorTypeInternalServer, "failed to read updated health record: "+err.Error()))
			return
		}

		result := HealthRecordResult{
			Records: []models.HealthRecord{*updatedRecord},
		}
		h.sendJSONResponse(w, result, http.StatusOK)
	}
}

// DeleteHealthRecord handles the deletion of a health record
func (h *HealthRecordHandler) DeleteHealthRecord(w http.ResponseWriter, r *http.Request) {
	// Set a timeout for the request context
	ctx, cancel := context.WithTimeout(r.Context(), time.Duration(config.RequestTimeoutSecond)*time.Second)
	defer cancel()

	// Get date from query parameters and parse it
	dateStr := r.URL.Query().Get("date")
	if dateStr == "" {
		h.handleError(w, apperr.NewAppError(apperr.ErrorTypeBadRequest, "date parameter is required"))
		return
	}

	date, err := time.Parse("20060102", dateStr)
	if err != nil {
		h.handleError(w, apperr.NewAppError(apperr.ErrorTypeInvalidDate, "Invalid date format: "+dateStr+" (Use YYYYMMDD)"))
		return
	}

	// Delete the record
	if err = h.DB.DeleteHealthRecord(ctx, date); err != nil {
		h.handleError(w, apperr.NewAppError(apperr.ErrorTypeInternalServer, "failed to delete health record: "+err.Error()))
		return
	}

	// Send success response
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Health record deleted successfully"})
}

// getByDate retrieves a record for the specified date (YYYYMMDD)
func (h *HealthRecordHandler) getByDate(ctx context.Context, dateStr string) (*models.HealthRecord, error) {
	date, err := time.Parse("20060102", dateStr)
	if err != nil {
		return nil, apperr.NewAppError(apperr.ErrorTypeInvalidDate, "Invalid date format: "+dateStr+" (Use YYYYMMDD)")
	}

	record, err := h.DB.ReadHealthRecord(ctx, date)
	if err != nil {
		return nil, apperr.NewAppError(apperr.ErrorTypeInternalServer, "Failed to read health record: "+err.Error())
	}
	if record == nil {
		return nil, apperr.NewAppError(apperr.ErrorTypeNotFound, "Unexpected: Health record not found for date: "+dateStr)
	}

	return record, nil
}

// getByYearMonth retrieves record(s) for the specified year and month (YYYY, MM)
func (h *HealthRecordHandler) getByYearMonth(ctx context.Context, yearStr, monthStr string) ([]models.HealthRecord, error) {
	year, err := time.Parse("2006", yearStr)
	if err != nil {
		return nil, apperr.NewAppError(apperr.ErrorTypeInvalidYear, "Invalid year format: "+yearStr+" (Use YYYY)")
	}

	if monthStr == "" {
		records, err := h.DB.ReadHealthRecordsByYear(ctx, year.Year())
		if err != nil {
			return nil, apperr.NewAppError(apperr.ErrorTypeInternalServer, "Failed to read health records: "+err.Error())
		}
		return records, nil
	}

	month, err := time.Parse("01", monthStr)
	if err != nil {
		return nil, apperr.NewAppError(apperr.ErrorTypeInvalidMonth, "Invalid month format: "+monthStr+" (Use MM)")
	}
	records, err := h.DB.ReadHealthRecordsByYearMonth(ctx, year.Year(), int(month.Month()))
	if err != nil {
		return nil, apperr.NewAppError(apperr.ErrorTypeInternalServer, "Failed to read  health records: "+err.Error())
	}

	return records, nil
}

// handleError processes errors and sends appropriate responses
func (h *HealthRecordHandler) handleError(w http.ResponseWriter, err error) {
	var appErr apperr.AppError
	if errors.As(err, &appErr) {

		log.Printf("application error: %v, Type: %s", appErr, appErr.Type)

		clientMessage := appErr.Error()

		if !config.IsDevelopment && appErr.Type == apperr.ErrorTypeInternalServer {
			clientMessage = "an internal server error occurred"
		}

		statusCode := http.StatusInternalServerError
		switch appErr.Type {
		case apperr.ErrorTypeInvalidDate, apperr.ErrorTypeInvalidYear, apperr.ErrorTypeInvalidMonth, apperr.ErrorTypeInvalidFormat, apperr.ErrorTypeBadRequest:
			statusCode = http.StatusBadRequest
		case apperr.ErrorTypeNotFound:
			statusCode = http.StatusNotFound
		}

		h.sendErrorResponse(w, apperr.AppError{Type: appErr.Type, Message: clientMessage}, statusCode)
	} else {
		log.Printf("unhandled error: %v", err)
		message := "an unexpected error occurred"
		if config.IsDevelopment {
			message = err.Error()
		}
		h.sendErrorResponse(w, apperr.NewAppError(apperr.ErrorTypeInternalServer, message), http.StatusInternalServerError)
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
