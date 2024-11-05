package apperr

type ErrorType string

const (
	ErrorTypeBadRequest     ErrorType = "BadRequest"
	ErrorTypeInvalidDate    ErrorType = "InvalidDate"
	ErrorTypeInvalidYear    ErrorType = "InvalidYear"
	ErrorTypeInvalidMonth   ErrorType = "InvalidMonth"
	ErrorTypeInvalidFormat  ErrorType = "InvalidFormat"
	ErrorTypeNotFound       ErrorType = "NotFound"
	ErrorTypeInternalServer ErrorType = "InternalServer"
)

type AppError struct {
	Type    ErrorType
	Message string
}

func (e AppError) Error() string {
	return e.Message
}

// NewAppError creates a new AppError with specified type and message
func NewAppError(errorType ErrorType, message string) AppError {
	return AppError{
		Type:    errorType,
		Message: message,
	}
}
