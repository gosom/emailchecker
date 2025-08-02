package handlers

import (
	"net/http"
	"net/url"

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
	email, err := url.QueryUnescape(chi.URLParam(r, "email"))
	if err != nil {
		return nil, errorsext.BadRequest("Invalid email parameter: failed to decode URL")
	}

	params := emailchecker.EmailCheckParams{
		Email: email,
	}

	result, err := h.checker.Check(r.Context(), params)
	if err != nil {
		aerr := errorsext.InternalServerError("Failed to check email", err)

		return nil, aerr
	}

	return result, nil
}
