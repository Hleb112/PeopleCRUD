package errors

import (
	"net/http"
)

type AppError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

func (e *AppError) Error() string {
	return e.Message
}

func NewAppError(code int, message, details string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Details: details,
	}
}

func NewBadRequestError(message string) *AppError {
	return NewAppError(http.StatusBadRequest, message, "")
}

func NewNotFoundError(message string) *AppError {
	return NewAppError(http.StatusNotFound, message, "")
}

func NewInternalServerError(message string) *AppError {
	return NewAppError(http.StatusInternalServerError, message, "")
}

func NewValidationError(details string) *AppError {
	return NewAppError(http.StatusBadRequest, "Validation failed", details)
}
