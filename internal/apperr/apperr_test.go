package apperr

import "testing"

func TestAppError_Error(t *testing.T) {
	err := AppError{Type: ErrorTypeNotFound, Message: "Record not found"}
	expectd := "Record not found"
	if got := err.Error(); got != expectd {
		t.Errorf("AppError.Ereror() = %v, want %v", got, expectd)
	}
}

func TestNewAppError(t *testing.T) {
	tests := []struct {
		name     string
		errType  ErrorType
		message  string
		expected AppError
	}{
		{
			name:     "Invalid Date Error",
			errType:  ErrorTypeInvalidDate,
			message:  "Invalid date format",
			expected: AppError{Type: ErrorTypeInvalidDate, Message: "Invalid date format"},
		},
		{
			name:     "Intenal Server Error",
			errType:  ErrorTypeInternalServer,
			message:  "Internal server error",
			expected: AppError{Type: ErrorTypeInternalServer, Message: "Internal server error"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewAppError(tt.errType, tt.message)
			if got.Type != tt.expected.Type || got.Message != tt.expected.Message {
				t.Errorf("NewAppError() = %v, want %v", got, tt.expected)
			}
		})
	}
}
