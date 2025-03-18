package mock

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/nnamm/go-health-tracker/internal/models"
)

var (
	ErrDataBaseConnection = errors.New("database connection failed")
	ErrDuplicateRecord    = errors.New("record already exists for date")
	ErrRecordNotFound     = errors.New("record not found")
	ErrTransactionFailed  = errors.New("transaction failed")
)

type MockDB struct {
	mu                sync.RWMutex
	records           map[time.Time]*models.HealthRecord
	createFunc        func(context.Context, *models.HealthRecord) (*models.HealthRecord, error)
	readFunc          func(context.Context, time.Time) (*models.HealthRecord, error)
	readYearFunc      func(context.Context, int) ([]models.HealthRecord, error)
	readYearMonthFunc func(context.Context, int, int) ([]models.HealthRecord, error)
	// updateFunc            func(*models.HealthRecord) error
	// deleteFunc            func(time.Time) error
	simulateTimeout       bool
	simulateContextCancel bool
	simulateDBError       bool
}

func NewMockDB() *MockDB {
	return &MockDB{
		records: make(map[time.Time]*models.HealthRecord),
	}
}

func (m *MockDB) SetSimulateTimeout(simulate bool) {
	m.simulateTimeout = simulate
}

func (m *MockDB) SetSimulateContextCalcel(simulate bool) {
	m.simulateContextCancel = simulate
}

func (m *MockDB) SetSimulateDBError(simulate bool) {
	m.simulateDBError = simulate
}

func (m *MockDB) checkContext() error {
	if m.simulateTimeout {
		return context.DeadlineExceeded
	}
	if m.simulateContextCancel {
		return context.Canceled
	}
	if m.simulateDBError {
		return ErrDataBaseConnection
	}
	return nil
}

func normalizeDate(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// CreateHealthRecord creates a new health record in the database
func (m *MockDB) CreateHealthRecord(ctx context.Context, hr *models.HealthRecord) (*models.HealthRecord, error) {
	if err := m.checkContext(); err != nil {
		return nil, err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.createFunc != nil {
		return m.createFunc(ctx, hr)
	}

	normalizedDate := normalizeDate(hr.Date)
	if normalizedDate.IsZero() {
		return nil, errors.New("date is required")
	}

	if _, exists := m.records[hr.Date]; exists {
		return nil, ErrDuplicateRecord
	}

	record := &models.HealthRecord{
		ID:        int64(len(m.records) + 1),
		Date:      normalizedDate,
		StepCount: hr.StepCount,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	m.records[normalizedDate] = record
	return record, nil
}

// ReadHealthRecord reads a health record from the database
func (m *MockDB) ReadHealthRecord(ctx context.Context, date time.Time) (*models.HealthRecord, error) {
	if err := m.checkContext(); err != nil {
		return nil, err
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.readFunc != nil {
		return m.readFunc(ctx, date)
	}

	normalizedDate := normalizeDate(date)
	record, exists := m.records[normalizedDate]
	if !exists {
		return nil, nil
	}
	return record, nil
}

// ReadHealthRecordsByYear reads health records from the database for a given year
func (m *MockDB) ReadHealthRecordsByYear(ctx context.Context, year int) ([]models.HealthRecord, error) {
	if err := m.checkContext(); err != nil {
		return nil, err
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.readYearFunc != nil {
		return m.readYearFunc(ctx, year)
	}

	if len(m.records) == 0 {
		return nil, nil
	}

	var records []models.HealthRecord
	for _, record := range m.records {
		if record.Date.Year() == year {
			records = append(records, *record)
		}
	}
	return records, nil
}

// ReadHealthRecordsByYearMonth reads health records from the database for a given year and month
func (m *MockDB) ReadHealthRecordsByYearMonth(ctx context.Context, year, month int) ([]models.HealthRecord, error) {
	if err := m.checkContext(); err != nil {
		return nil, err
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.readYearMonthFunc != nil {
		return m.readYearMonthFunc(ctx, year, month)
	}

	if len(m.records) == 0 {
		return nil, nil
	}

	var records []models.HealthRecord
	for _, record := range m.records {
		if record.Date.Year() == year && record.Date.Month() == time.Month(month) {
			records = append(records, *record)
		}
	}
	return records, nil
}
