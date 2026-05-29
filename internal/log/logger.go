package log

import (
	"log/slog"
	"os"
	"strings"
)

// Setup 根据环境和日志级别配置全局 slog logger。
// local: 文本格式输出到 stderr
// production/test: JSON 格式输出到 stdout
func Setup(env, level string) {
	var handler slog.Handler

	opts := &slog.HandlerOptions{
		Level: parseLevel(level),
	}

	if env == "local" {
		handler = slog.NewTextHandler(os.Stderr, opts)
	} else {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}

	logger := slog.New(handler).With(
		slog.String("app", "nucleus"),
		slog.String("env", env),
	)

	slog.SetDefault(logger)
}

func parseLevel(s string) slog.Level {
	switch strings.ToLower(s) {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
