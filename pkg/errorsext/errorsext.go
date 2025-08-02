package errorsext

import (
	"fmt"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/cockroachdb/errors/errbase"
)

func WithStackTrace(err error, depth ...int) error {
	if err == nil {
		return nil
	}

	if HasStackTrace(err) {
		return err
	}

	if len(depth) > 0 {
		return errors.WithStackDepth(err, depth[0])
	}

	return errors.WithStackDepth(err, 1)
}

type stackTracer interface {
	StackTrace() errbase.StackTrace
}

func HasStackTrace(err error) bool {
	if err == nil {
		return false
	}

	_, found := errors.If(err, func(err error) (any, bool) {
		_, ok := err.(stackTracer)
		return nil, ok
	})

	return found
}

func FormatStackTrace(err error) string {
	var st stackTracer
	if !errors.As(err, &st) {
		return ""
	}

	trace := st.StackTrace()
	if trace == nil {
		return ""
	}

	var builder strings.Builder

	maxFrames := 5
	if len(trace) > maxFrames {
		trace = trace[:maxFrames]
	}

	for _, frame := range trace {
		frameStr := fmt.Sprintf("%+s", frame)
		parts := strings.SplitN(frameStr, "\n\t", 2)
		if len(parts) != 2 {
			continue
		}

		functionName := parts[0]
		filePath := parts[1]
		lineStr := fmt.Sprintf("%d", frame)

		builder.WriteString(functionName)
		builder.WriteString("\n\t")
		builder.WriteString(fmt.Sprintf("%s:%s", filePath, lineStr))
		builder.WriteString("\n")
	}

	return builder.String()
}
