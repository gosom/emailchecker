package httpext

import (
	"io"
	"net/http"
	"strings"
)

type ResponseCapture struct {
	http.ResponseWriter
	HandlerError       error
	StatusCode         int
	bodyWriter         io.Writer
	BodyCapture        io.Writer
	contentTypeChecked bool
	wroteHeader        bool
}

func (rc *ResponseCapture) SetHandlerError(err error) {
	rc.HandlerError = err
}

func (rc *ResponseCapture) Write(b []byte) (int, error) {
	if rc.StatusCode == 0 {
		rc.WriteHeader(http.StatusOK)
	}

	if rc.bodyWriter != nil {
		n, err := rc.bodyWriter.Write(b)
		return n, err
	}

	return rc.ResponseWriter.Write(b)
}

func (rc *ResponseCapture) WriteHeader(statusCode int) {
	defer func() {
		if !rc.wroteHeader {
			rc.wroteHeader = true
		}
	}()

	if !rc.wroteHeader {
		rc.StatusCode = statusCode
	}

	if !rc.contentTypeChecked && rc.BodyCapture != nil {
		rc.contentTypeChecked = true
		contentType := rc.Header().Get("Content-Type")

		if rc.BodyCapture != nil && IsLoggableContentType(contentType) {
			rc.bodyWriter = io.MultiWriter(rc.ResponseWriter, rc.BodyCapture)
		} else {
			rc.bodyWriter = rc.ResponseWriter
			rc.BodyCapture = nil
		}
	}

	if !rc.wroteHeader {
		rc.ResponseWriter.WriteHeader(statusCode)
	}
}

func IsLoggableContentType(contentType string) bool {
	contentType = strings.ToLower(contentType)

	prefixes := []string{
		"application/json",
		"application/x-www-form-urlencoded",
	}

	for i := range prefixes {
		if strings.HasPrefix(contentType, prefixes[i]) {
			return true
		}
	}

	return false
}
