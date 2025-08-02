package log

import (
	"context"
	"encoding/base32"
	"encoding/binary"
	"log/slog"
	"maps"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"emailchecker/pkg/errorsext"
)

type key int

const loggerKey key = 0

var (
	base *slog.Logger
	once sync.Once
)

type Logger struct {
	mu     sync.RWMutex
	fields map[string]any
}

func init() {
	base = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
}

func Init(debug bool, args ...any) {
	once.Do(func() {
		level := slog.LevelInfo
		if debug {
			level = slog.LevelDebug
		}

		base = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: level,
		})).With(args...)
	})
}

func New(ctx context.Context) context.Context {
	return context.WithValue(ctx, loggerKey, &Logger{
		fields: make(map[string]any, 8),
	})
}

func Set(ctx context.Context, k string, v any) {
	if logger, ok := ctx.Value(loggerKey).(*Logger); ok {
		logger.mu.Lock()
		logger.fields[k] = v
		logger.mu.Unlock()
	}
}

func MapSet(ctx context.Context, fields map[string]any) {
	if logger, ok := ctx.Value(loggerKey).(*Logger); ok {
		logger.mu.Lock()
		maps.Copy(logger.fields, fields)
		logger.mu.Unlock()
	}
}

func Debug(ctx context.Context, msg string, args ...any) {
	logWithContext(ctx, slog.LevelDebug, msg, args...)
}

func Info(ctx context.Context, msg string, args ...any) {
	logWithContext(ctx, slog.LevelInfo, msg, args...)
}

func Warn(ctx context.Context, msg string, args ...any) {
	logWithContext(ctx, slog.LevelWarn, msg, args...)
}

func Error(ctx context.Context, err error) {
	var args []any

	if stackTrace := errorsext.FormatStackTrace(err); stackTrace != "" {
		args = append(args, slog.String("stacktrace", stackTrace))
	}

	logWithContext(ctx, slog.LevelError, err.Error(), args...)
}

func ErrorWithMessage(ctx context.Context, msg string, err error) {
	if err == nil {
		logWithContext(ctx, slog.LevelError, msg)

		return
	}

	var args []any

	if stackTrace := errorsext.FormatStackTrace(err); stackTrace != "" {
		args = append(args, slog.Any("stacktrace", stackTrace))
	}

	logWithContext(ctx, slog.LevelError, msg, append(args, slog.String("error", err.Error()))...)
}

func logWithContext(ctx context.Context, level slog.Level, msg string, args ...any) {
	if logger, ok := ctx.Value(loggerKey).(*Logger); ok {
		logger.mu.RLock()

		if len(logger.fields) == 0 {
			logger.mu.RUnlock()
			base.Log(ctx, level, msg, args...)
			return
		}

		allArgs := make([]any, 0, len(logger.fields)*2+len(args))
		for k, v := range logger.fields {
			allArgs = append(allArgs, k, v)
		}
		logger.mu.RUnlock()

		allArgs = append(allArgs, args...)
		base.Log(ctx, level, msg, allArgs...)
	} else {
		base.Log(ctx, level, msg, args...)
	}
}

var (
	counter  uint64
	encoding = base32.NewEncoding("0123456789abcdefghijklmnopqrstuv").WithPadding(base32.NoPadding)
)

func ID() string {
	now := time.Now().UnixNano()
	seq := atomic.AddUint64(&counter, 1)

	combined := uint64(now) ^ (seq << 32) //nolint:gosec // it's fine

	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], combined)

	return encoding.EncodeToString(buf[:])[:10]
}
