package logger

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"
)

// カスタムログレベル
const (
	LevelTrace slog.Level = -8
	LevelDebug slog.Level = slog.LevelDebug // -4
	LevelInfo  slog.Level = slog.LevelInfo  // 0
	LevelWarn  slog.Level = slog.LevelWarn  // 4
	LevelError slog.Level = slog.LevelError // 8
	LevelFatal slog.Level = 12
)

// Config はロガーの設定。
type Config struct {
	Level       string // "trace", "debug", "info", "warn", "error", "fatal"
	ServiceName string
	Version     string
	Environment string
}

// New はプラン仕様に準拠したslogロガーを作成する。
func New(cfg Config) *slog.Logger {
	level := parseLevel(cfg.Level)

	opts := &slog.HandlerOptions{
		Level: level,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// キー名の変更
			switch a.Key {
			case slog.TimeKey:
				a.Key = "timestamp"
			case slog.MessageKey:
				a.Key = "message"
			case slog.LevelKey:
				// カスタムレベル名
				if lvl, ok := a.Value.Any().(slog.Level); ok {
					a.Value = slog.StringValue(levelName(lvl))
				}
			}
			// WARN以上でソース情報を出力
			return a
		},
		AddSource: level.Level() <= LevelWarn,
	}

	handler := slog.NewJSONHandler(os.Stdout, opts)

	return slog.New(handler).With(
		"service.name", cfg.ServiceName,
		"service.version", cfg.Version,
		"deployment.environment", cfg.Environment,
	)
}

// PrintBanner は起動時バナーを標準出力に出力する。
func PrintBanner(cfg Config, address string, scyllaHost string) {
	fmt.Println("==================================================")
	fmt.Printf(" Service:   %s\n", cfg.ServiceName)
	fmt.Printf(" Ver:       %s\n", cfg.Version)
	fmt.Printf(" Timestamp: %s\n", time.Now().Format(time.RFC3339))
	fmt.Printf(" ENV:       %s\n", cfg.Environment)
	fmt.Printf(" LogLevel:  %s\n", cfg.Level)
	fmt.Printf(" Address:   %s\n", address)
	if scyllaHost != "" {
		fmt.Printf(" Scylla:    %s\n", scyllaHost)
	}
	fmt.Println("==================================================")
}

func parseLevel(s string) slog.Level {
	switch strings.ToLower(s) {
	case "trace":
		return LevelTrace
	case "debug":
		return LevelDebug
	case "warn":
		return LevelWarn
	case "error":
		return LevelError
	case "fatal":
		return LevelFatal
	default:
		return LevelInfo
	}
}

func levelName(l slog.Level) string {
	switch {
	case l >= LevelFatal:
		return "FATAL"
	case l >= LevelError:
		return "ERROR"
	case l >= LevelWarn:
		return "WARN"
	case l >= LevelInfo:
		return "INFO"
	case l >= LevelDebug:
		return "DEBUG"
	default:
		return "TRACE"
	}
}
