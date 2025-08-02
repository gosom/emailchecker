package handlers

import (
	"net/http"

	"emailchecker"

	"github.com/go-chi/chi/v5"

	"emailchecker/pkg/errorsext"
)

type CheckHandler struct {
	checker *emailchecker.EmailChecker
}

func NewCheckHandler(checker *emailchecker.EmailChecker) *CheckHandler {
	return &CheckHandler{
		checker: checker,
	}
}

func (h *CheckHandler) CheckEmail(_ http.ResponseWriter, r *http.Request) (any, *errorsext.APIError) {
	params := emailchecker.EmailCheckParams{
		Email: chi.URLParam(r, "email"),
	}

	result, err := h.checker.Check(r.Context(), params)
	if err != nil {
		aerr := errorsext.InternalServerError("Failed to check email", err)

		return nil, aerr
	}

	return result, nil
}
