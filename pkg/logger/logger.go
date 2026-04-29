package logger

import (
	"io"
	"log/slog"
	"os"
	"strings"
)

// New creates a slog.Logger that writes to both stdout and a log file.
// level: "debug" | "info" | "warn" | "error"
func New(level, filePath string) (*slog.Logger, func(), error) {
	var lvl slog.Level
	switch strings.ToLower(level) {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}

	writers := []io.Writer{os.Stdout}
	var closeFile func()

	if filePath != "" {
		f, err := os.OpenFile(filePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			return nil, nil, err
		}
		writers = append(writers, f)
		closeFile = func() { f.Close() }
	} else {
		closeFile = func() {}
	}

	mw := io.MultiWriter(writers...)
	handler := slog.NewJSONHandler(mw, &slog.HandlerOptions{Level: lvl})
	return slog.New(handler), closeFile, nil
}
