package handlers

import (
	"net/http"

	"emailchecker/pkg/errorsext"
	"emailchecker/pkg/httpext"
)

type OpsHandler struct{}

func NewOpsHandler() *OpsHandler {
	return &OpsHandler{}
}

func (h *OpsHandler) Health(_ http.ResponseWriter, r *http.Request) (any, *errorsext.APIError) {
	httpext.SetStatusCode(r, http.StatusNoContent)

	return nil, nil
}

func (h *OpsHandler) NotFound(_ http.ResponseWriter, r *http.Request) (any, *errorsext.APIError) {
	httpext.SetStatusCode(r, http.StatusNotFound)

	return nil, errorsext.NotFound("not found")
}

func (h *OpsHandler) MethodNotAllowed(_ http.ResponseWriter, r *http.Request) (any, *errorsext.APIError) {
	httpext.SetStatusCode(r, http.StatusMethodNotAllowed)

	return nil, errorsext.MethodNotAllowed("not allowed")
}
