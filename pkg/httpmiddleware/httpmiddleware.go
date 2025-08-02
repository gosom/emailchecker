package httpmiddleware

import (
	"encoding/json"
	"fmt"
	"net/http"

	"emailchecker/pkg/errorsext"
	"emailchecker/pkg/httpext"
	"emailchecker/pkg/log"
)

func Handler(fn func(http.ResponseWriter, *http.Request) (any, *errorsext.APIError)) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				err := errorsext.WithStackTrace(fmt.Errorf("panic: %v", rec), 3)

				panicErr := errorsext.InternalServerError("internal server error", err)
				JSON(w, r, nil, panicErr)
			}
		}()

		result, err := fn(w, r)
		if err != nil {
			JSON(w, r, nil, err)

			return
		}

		JSON(w, r, result, nil)
	})
}

func JSON(w http.ResponseWriter, r *http.Request, data any, apiErr *errorsext.APIError) {
	w.Header().Set("Content-Type", "application/json")

	enc := json.NewEncoder(w)

	if apiErr != nil {
		w.WriteHeader(apiErr.StatusCode)
		if apiErr.InternalError != nil {
			if rw, ok := w.(interface{ SetHandlerError(error) }); ok {
				rw.SetHandlerError(apiErr.InternalError)
			}
		}

		if err := enc.Encode(apiErr); err != nil {
			werr := errorsext.WithStackTrace(fmt.Errorf("failed to encode API error response: %w", err))
			log.Error(r.Context(), werr)
		}

		return
	}

	w.WriteHeader(httpext.GetStatusCode(r))

	if data != nil {
		if err := enc.Encode(data); err != nil {
			werr := errorsext.WithStackTrace(fmt.Errorf("failed to encode response: %w", err))
			log.Error(r.Context(), werr)
		}
	}
}
