// Package errors provides custom error types and error handling utilities.
//
// This package defines application-specific error types that can be used
// throughout the codebase for consistent error handling and reporting.
package errors

// AppError is the foundation for all custom errors in this project.
type AppError struct {
	Message string
	Code    string
}

// Error implements the error interface.
func (e *AppError) Error() string {
	return e.Message
}

// New creates a new AppError with the given message and code.
func New(message, code string) *AppError {
	return &AppError{
		Message: message,
		Code:    code,
	}
}
