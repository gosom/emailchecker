package httpmiddleware

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/jonboulle/clockwork"

	"emailchecker/pkg/httpext"
	"emailchecker/pkg/log"
)

func Logging(opts ...LoggingOption) func(http.Handler) http.Handler {
	config := &loggingConfig{
		maxBodySize:      1024 * 20, // 20 KB
		clock:            clockwork.NewRealClock(),
		logRequestBody:   true,
		logResponseBody:  false,
		enableMetrics:    true,
		headersWhiteList: []string{"User-Agent", "Authorization", "Origin", "Referer", "Content-Type"},
	}

	for _, opt := range opts {
		opt(config)
	}

	logAllHeaders := len(config.headersWhiteList) == 0
	headersToLog := make(map[string]bool)
	for _, header := range config.headersWhiteList {
		headersToLog[strings.ToLower(header)] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			startTime := config.clock.Now()
			ctx := log.New(r.Context())
			correlationID := r.Header.Get("X-Correlation-ID")
			if correlationID == "" {
				correlationID = log.ID()
			}

			headers := getHeaderMap()
			defer putHeaderMap(headers)

			for k, v := range r.Header {
				if logAllHeaders || headersToLog[strings.ToLower(k)] {
					if len(v) == 1 {
						headers[k] = v[0]
					} else {
						headers[k] = strings.Join(v, ", ")
					}

					if strings.EqualFold(k, "Authorization") {
						// Mask the Authorization header value
						headers[k] = "REDACTED"
					} else if strings.EqualFold(k, "Cookie") {
						// Mask the Cookie header value
						headers[k] = "REDACTED"
					}
				}
			}
			requestMap := getStringAnyMap()
			defer putStringAnyMap(requestMap)

			requestMap["request_id"] = log.ID()
			requestMap["method"] = r.Method
			requestMap["path"] = r.URL.Path
			requestMap["query_string"] = r.URL.RawQuery
			requestMap["remote_addr"] = r.RemoteAddr
			requestMap["headers"] = headers

			var bodyCapture *portionCapture

			contentType := r.Header.Get("Content-Type")
			if config.logRequestBody && r.Body != nil && httpext.IsLoggableContentType(contentType) {
				// bodyCapture will be populated when the r.Body is read in the
				// handler, capturing only the first `maxBodySize` bytes of the body.
				// This means that the body will not be read until the handler
				// processes the request
				bodyCapture = newPortionCapture(config.maxBodySize)
				defer bodyCapture.Release()
				r.Body = io.NopCloser(io.TeeReader(r.Body, bodyCapture))
			}

			log.MapSet(ctx, map[string]any{
				"correlation_id": correlationID,
				"request":        requestMap,
			})

			rw := httpext.ResponseCapture{
				ResponseWriter: w,
			}

			var responseCapture *portionCapture
			if config.logResponseBody {
				responseCapture = newPortionCapture(config.maxBodySize)
				defer responseCapture.Release()
				var capturer io.Writer = responseCapture
				rw.BodyCapture = capturer
			}

			log.MapSet(ctx, map[string]any{
				"correlation_id": correlationID,
				"request_id":     requestMap["request_id"],
			})

			r = r.WithContext(ctx)

			next.ServeHTTP(&rw, r)

			duration := config.clock.Since(startTime)

			if rw.StatusCode >= 400 && bodyCapture != nil {
				if bodyCapture.HasMore() {
					requestMap["body"] = bodyCapture.buffer.String() + "... (truncated)"
				} else {
					requestMap["body"] = bodyCapture.buffer.String()
				}
			}

			responseMap := getStringAnyMap()
			defer putStringAnyMap(responseMap)

			if config.logResponseHeaders {
				responseHeaders := getHeaderMap()
				for k, v := range rw.Header() {
					if len(v) == 1 {
						responseHeaders[k] = v[0]
					} else {
						responseHeaders[k] = strings.Join(v, ", ")
					}
				}

				responseMap["headers"] = responseHeaders
				defer putHeaderMap(responseHeaders)
			}

			if rw.BodyCapture != nil && responseCapture != nil {
				if responseCapture.HasMore() {
					responseMap["body"] = responseCapture.buffer.String() + "... (truncated)"
				} else {
					responseMap["body"] = responseCapture.buffer.String()
				}
			}

			responseMap["status_code"] = rw.StatusCode
			responseMap["latency_ms"] = float64(duration.Nanoseconds()) / 1e6 // Convert to milliseconds

			log.Set(ctx, "response", responseMap)

			switch {
			case rw.StatusCode >= 0 && rw.StatusCode < 400:
				log.Info(ctx, r.URL.String())
			case rw.StatusCode >= 400 && rw.StatusCode < 500:
				log.Warn(ctx, r.URL.String())
			default:
				log.ErrorWithMessage(ctx, r.URL.String(), rw.HandlerError)
			}
		})
	}
}

func SkipRequestBodyLogging() LoggingOption {
	return func(c *loggingConfig) {
		c.logRequestBody = false
	}
}

func LogResponseBody() LoggingOption {
	return func(c *loggingConfig) {
		c.logResponseBody = true
	}
}

func LogMaxBodySize(size int64) LoggingOption {
	return func(c *loggingConfig) {
		c.maxBodySize = size
	}
}

func LogResponseHeaders() LoggingOption {
	return func(c *loggingConfig) {
		c.logResponseHeaders = true
	}
}

func DisableMetrics() LoggingOption {
	return func(c *loggingConfig) {
		c.enableMetrics = false
	}
}

func HeadersWhiteList(whitelist ...string) LoggingOption {
	return func(c *loggingConfig) {
		c.headersWhiteList = whitelist
	}
}

// ----- Helpers -----

type loggingConfig struct {
	clock              clockwork.Clock
	logRequestBody     bool
	logResponseBody    bool
	logResponseHeaders bool
	maxBodySize        int64
	enableMetrics      bool
	headersWhiteList   []string // Headers to log, if empty all headers are logged
}

type LoggingOption func(*loggingConfig)

// Custom writer that only captures a specific portion
type portionCapture struct {
	buffer  *bytes.Buffer
	length  int64 // Capture this many bytes
	written int64 // Total bytes written so far
}

func newPortionCapture(length int64) *portionCapture {
	return &portionCapture{
		buffer: bufferPool.Get().(*bytes.Buffer), //nolint:errcheck // its fine
		length: length,
	}
}

func (pc *portionCapture) HasMore() bool {
	return pc.written > pc.length
}

func (pc *portionCapture) Write(p []byte) (n int, err error) {
	n = len(p)

	// Since start=0, this can be much simpler:
	remaining := pc.length - int64(pc.buffer.Len())
	if remaining > 0 {
		toCopy := min(int64(len(p)), remaining)
		pc.buffer.Write(p[:toCopy])
	}

	pc.written += int64(n)
	return n, nil
}

func (pc *portionCapture) Release() {
	pc.buffer.Reset()
	bufferPool.Put(pc.buffer)
}

var (
	stringAnyMapPool = sync.Pool{
		New: func() any {
			return make(map[string]any, 8)
		},
	}

	headerMapPool = sync.Pool{
		New: func() any {
			return make(map[string]string, 16)
		},
	}

	bufferPool = sync.Pool{
		New: func() any {
			return &bytes.Buffer{}
		},
	}
)

func getStringAnyMap() map[string]any {
	return stringAnyMapPool.Get().(map[string]any) //nolint:errcheck // its fine
}

func putStringAnyMap(m map[string]any) {
	for k := range m {
		delete(m, k)
	}

	stringAnyMapPool.Put(m)
}

func getHeaderMap() map[string]string {
	return headerMapPool.Get().(map[string]string) //nolint:errcheck // its fine
}

func putHeaderMap(m map[string]string) {
	for k := range m {
		delete(m, k)
	}

	headerMapPool.Put(m)
}
