package validators

import (
	"github.com/nnamm/go-health-tracker/internal/apperr"
	"github.com/nnamm/go-health-tracker/internal/models"
)

type HealthRecordValidator interface {
	Validate(*models.HealthRecord) error
}

type DefaultHealthRecordValidator struct{}

func NewHealthRecordValidator() HealthRecordValidator {
	return &DefaultHealthRecordValidator{}
}

func (v *DefaultHealthRecordValidator) Validate(hr *models.HealthRecord) error {
	if hr == nil {
		return apperr.NewAppError(apperr.ErrorTypeInvalidFormat, "health record is required")
	}

	if hr.Date.IsZero() {
		return apperr.NewAppError(apperr.ErrorTypeInvalidDate, "date is required")
	}

	if hr.StepCount < 0 {
		return apperr.NewAppError(apperr.ErrorTypeInvalidFormat, "step count must not be negative")
	}

	return nil
}
