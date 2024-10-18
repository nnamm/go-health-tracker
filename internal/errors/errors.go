package errors

type ErrorType string

const (
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

// func NewInvalidDateError(date string) AppError {
// 	return AppError{
// 		Type:    ErrorTypeInvalidDate,
// 		Message: fmt.Sprintf("Invalid date format: %s. Use YYYYMMDD", date),
// 	}
// }
//
// func NewInvalidYearError(year string) AppError {
// 	return AppError{
// 		Type:    ErrorTypeInvalidYear,
// 		Message: fmt.Sprintf("Invalid year format: %s. Use YYYY", year),
// 	}
// }
//
// func NewInvalidMonthError(month string) AppError {
// 	return AppError{
// 		Type:    ErrorTypeInvalidMonth,
// 		Message: fmt.Sprintf("Invalid month format: %s. Use MM", month),
// 	}
// }
//
// func NewInvalidFormatError(format, expected string) AppError {
// 	return AppError{
// 		Type:    ErrorTypeInvalidFormat,
// 		Message: fmt.Sprintf("Invalid format: %s. Expected: %s", format, expected),
// 	}
// }
//
// func NewNotFoundError(resource string) AppError {
// 	return AppError{
// 		Type:    ErrorTypeNotFound,
// 		Message: fmt.Sprintf("Resource not found: %s", resource),
// 	}
// }

// NewAppError creates a new AppError with specified type and message
func NewAppError(errorType ErrorType, message string) AppError {
	return AppError{
		Type:    errorType,
		Message: message,
	}
}
