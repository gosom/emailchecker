package errorsext

import (
	"fmt"
	"net/http"
)

type APIError struct {
	// StatusCode is the HTTP status code for the error.
	StatusCode       int               `json:"status_code"`
	Code             string            `json:"code"`
	Message          string            `json:"message"`
	ValidationErrors *ValidationErrors `json:"validation_errors,omitempty"`
	InternalError    error             `json:"-"`
}

func (e *APIError) Error() string {
	return fmt.Sprintf("APIError: %s (code: %s, status: %d)", e.Message, e.Code, e.StatusCode)
}

// some common API errors

func BadRequest(message string) *APIError {
	return &APIError{
		StatusCode: http.StatusBadRequest,
		Code:       "bad_request",
		Message:    message,
	}
}

func Unauthorized(message string) *APIError {
	return &APIError{
		StatusCode: 401,
		Code:       "unauthorized",
		Message:    message,
	}
}

func Forbidden(message string) *APIError {
	return &APIError{
		StatusCode: http.StatusForbidden,
		Code:       "forbidden",
		Message:    message,
	}
}

func NotFound(message string) *APIError {
	return &APIError{
		StatusCode: http.StatusNotFound,
		Code:       "not_found",
		Message:    message,
	}
}

func InternalServerError(message string, internalError error) *APIError {
	return &APIError{
		StatusCode:    http.StatusInternalServerError,
		Code:          "internal_error",
		Message:       message,
		InternalError: internalError,
	}
}

func Conflict(message string) *APIError {
	return &APIError{
		StatusCode: http.StatusConflict,
		Code:       "conflict",
		Message:    message,
	}
}

func UnprocessableEntity(message string, verrs *ValidationErrors) *APIError {
	return &APIError{
		StatusCode:       http.StatusUnprocessableEntity,
		Code:             "unprocessable_entity",
		Message:          message,
		ValidationErrors: verrs,
	}
}

func TooManyRequests(message string) *APIError {
	return &APIError{
		StatusCode: http.StatusTooManyRequests,
		Code:       "too_many_requests",
		Message:    message,
	}
}

func ServiceUnavailable(message string) *APIError {
	return &APIError{
		StatusCode: http.StatusServiceUnavailable,
		Code:       "service_unavailable",
		Message:    message,
	}
}

func NotImplemented(message string) *APIError {
	return &APIError{
		StatusCode: http.StatusNotImplemented,
		Code:       "not_implemented",
		Message:    message,
	}
}

func BadGateway(message string) *APIError {
	return &APIError{
		StatusCode: http.StatusBadGateway,
		Code:       "bad_gateway",
		Message:    message,
	}
}

func MethodNotAllowed(message string) *APIError {
	return &APIError{
		StatusCode: http.StatusMethodNotAllowed,
		Code:       "method_not_allowed",
		Message:    message,
	}
}
