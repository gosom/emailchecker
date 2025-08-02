package httpext

import (
	"bytes"
	"io"
	"net/http"

	"emailchecker/pkg/log"
)

func WrapClient(client *http.Client) *http.Client {
	transport := client.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}

	client.Transport = &loggingTransport{
		Transport: transport,
	}

	return client
}

type loggingTransport struct {
	Transport http.RoundTripper
}

func (lt *loggingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	msg := "external_request"
	args := []any{"method", req.Method, "url", req.URL.String()}
	fn := log.Debug

	defer func() {
		fn(req.Context(), msg, args...)
	}()

	var (
		reqBodyBytes  []byte
		respBodyBytes []byte
		err2          error
		debugLog      bool
	)

	if req.Body != nil {
		reqBodyBytes, err2 = io.ReadAll(req.Body)
		if err2 == nil {
			req.Body = io.NopCloser(bytes.NewBuffer(reqBodyBytes))
		}
	}

	resp, err := lt.Transport.RoundTrip(req)
	if err != nil {
		args = append(args, "error", err, "requestBody", string(reqBodyBytes))
		fn = log.Warn

		return resp, err
	}

	args = append(args, "statusCode", resp.StatusCode)
	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		fn = log.Warn
		debugLog = true
	}

	if debugLog && resp.Body != nil {
		respBodyBytes, err2 = io.ReadAll(resp.Body)
		if err2 == nil {
			resp.Body = io.NopCloser(bytes.NewBuffer(respBodyBytes))
		}
	}

	if debugLog {
		args = append(args, "responseBody", string(respBodyBytes), "requestBody", string(reqBodyBytes))
	}

	return resp, nil
}
