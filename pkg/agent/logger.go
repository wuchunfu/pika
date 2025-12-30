package agent

import (
	"io"
	"log/slog"
	"os"
	"strings"

	"gopkg.in/natefinch/lumberjack.v2"
)

// LogConfig 日志配置
type LogConfig struct {
	Level      string
	File       string
	MaxSize    int
	MaxBackups int
	MaxAge     int
	Compress   bool
}

// InitLogger 初始化日志系统
func InitLogger(cfg *LogConfig) {
	var level slog.Level
	switch strings.ToLower(cfg.Level) {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	var writer io.Writer

	// 如果配置了日志文件，使用 lumberjack 进行日志滚动
	if cfg.File != "" {
		writer = &lumberjack.Logger{
			Filename:   cfg.File,
			MaxSize:    cfg.MaxSize,    // MB
			MaxBackups: cfg.MaxBackups, // 保留的旧日志文件数
			MaxAge:     cfg.MaxAge,     // 天数
			Compress:   cfg.Compress,   // 是否压缩
		}
	} else {
		// 否则输出到标准输出
		writer = os.Stdout
	}

	// 创建文本格式的 handler（更易读）
	opts := &slog.HandlerOptions{
		Level: level,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// 格式化时间为更易读的格式
			if a.Key == slog.TimeKey {
				return slog.String(slog.TimeKey, a.Value.Time().Format("2006-01-02 15:04:05.000"))
			}
			return a
		},
	}
	handler := slog.NewTextHandler(writer, opts)
	logger := slog.New(handler)
	slog.SetDefault(logger)
}
