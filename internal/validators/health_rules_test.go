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
	now := time.Now()
	pastDate := now.AddDate(0, 0, -1)
	futureDate := now.AddDate(0, 0, 1)
	maxValidSteps := 100000

	tests := []struct {
		name      string
		record    *models.HealthRecord
		wantErr   bool
		errorType apperr.ErrorType
		errorMsg  string
	}{
		{
			name: "有効なレコード - 現在の日付",
			record: &models.HealthRecord{
				Date:      now,
				StepCount: 1000,
			},
			wantErr: false,
		},
		{
			name: "有効なレコード - 過去の日付",
			record: &models.HealthRecord{
				Date:      pastDate,
				StepCount: 1000,
			},
			wantErr: false,
		},
		{
			name:      "nilレコード",
			record:    nil,
			wantErr:   true,
			errorType: apperr.ErrorTypeInvalidFormat,
			errorMsg:  "health record is required",
		},
		{
			name: "ゼロ日付",
			record: &models.HealthRecord{
				StepCount: 1000,
			},
			wantErr:   true,
			errorType: apperr.ErrorTypeInvalidDate,
			errorMsg:  "date is required",
		},
		{
			name: "負の歩数",
			record: &models.HealthRecord{
				Date:      now,
				StepCount: -1,
			},
			wantErr:   true,
			errorType: apperr.ErrorTypeInvalidFormat,
			errorMsg:  "step count must not be negative",
		},
		{
			name: "歩数が上限を超えている",
			record: &models.HealthRecord{
				Date:      now,
				StepCount: maxValidSteps + 1,
			},
			wantErr:   true,
			errorType: apperr.ErrorTypeInvalidFormat,
			errorMsg:  "step count is unrealistically high",
		},
		{
			name: "未来の日付",
			record: &models.HealthRecord{
				Date:      futureDate,
				StepCount: 1000,
			},
			wantErr:   true,
			errorType: apperr.ErrorTypeInvalidDate,
			errorMsg:  "future dates are not allowed",
		},
		{
			name: "境界値 - 最大有効歩数",
			record: &models.HealthRecord{
				Date:      now,
				StepCount: maxValidSteps,
			},
			wantErr: false,
		},
		{
			name: "境界値 - ゼロ歩数",
			record: &models.HealthRecord{
				Date:      now,
				StepCount: 0,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.Validate(tt.record)
			if tt.wantErr {
				assert.Error(t, err)
				// check the type of AppError with type assertion
				if appErr, ok := err.(apperr.AppError); ok {
					assert.Equal(t, tt.errorType, appErr.Type)
					assert.Equal(t, tt.errorMsg, appErr.Message)
				} else {
					t.Errorf("expected apperr.AppError, got %T", err)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
