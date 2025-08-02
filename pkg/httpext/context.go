package httpext

import (
	"context"
	"net/http"
)

type contextKey string

const statusCodeKey contextKey = "statusCode"

func SetStatusCode(r *http.Request, statusCode int) {
	ctx := context.WithValue((*r).Context(), statusCodeKey, statusCode)
	*r = *r.WithContext(ctx)
}

func GetStatusCode(r *http.Request) int {
	statusCode, ok := r.Context().Value(statusCodeKey).(int)
	if !ok {
		return http.StatusOK
	}

	return statusCode
}
