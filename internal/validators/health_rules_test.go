package validators

import (
	"testing"
	"time"

	"github.com/nnamm/go-health-tracker/internal/apperr"
	"github.com/nnamm/go-health-tracker/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestDefaultHealthRecordValidator_Validate(t *testing.T) {
	v := NewHealthRecordValidator()

	tests := []struct {
		name      string
		record    *models.HealthRecord
		wantErr   bool
		errorType string
	}{
		{
			name: "valid record",
			record: &models.HealthRecord{
				Date:      time.Now(),
				StepCount: 1000,
			},
			wantErr: false,
		},
		{
			name:      "nil record",
			record:    nil,
			wantErr:   true,
			errorType: "InvalidFormat",
		},
		{
			name: "zero date",
			record: &models.HealthRecord{
				StepCount: 1000,
			},
			wantErr:   true,
			errorType: "InvalidDate",
		},
		{
			name: "nagarive step count",
			record: &models.HealthRecord{
				Date:      time.Now(),
				StepCount: -1,
			},
			wantErr:   true,
			errorType: "InvalidFormat",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.Validate(tt.record)
			if tt.wantErr {
				assert.Error(t, err)
				// check the type of AppError with type assertion
				if appErr, ok := err.(apperr.AppError); ok {
					assert.Equal(t, tt.errorType, string(appErr.Type))
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
