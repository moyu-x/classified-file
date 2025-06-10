package logger

import (
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var (
	// Logger 全局日志实例
	Logger zerolog.Logger
)

// Init 初始化日志配置
func Init(debug bool) {
	// 设置时间格式
	zerolog.TimeFieldFormat = time.RFC3339

	// 设置输出格式
	var output io.Writer = os.Stdout
	if !debug {
		// 在非调试模式下，使用格式化的控制台输出
		output = zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: "15:04:05",
		}
	}

	// 设置日志级别
	level := zerolog.InfoLevel
	if debug {
		level = zerolog.DebugLevel
	}

	// 初始化全局日志实例
	Logger = zerolog.New(output).
		Level(level).
		With().
		Timestamp().
		Caller().
		Logger()

	// 设置全局默认日志
	log.Logger = Logger
}

// Debug 输出调试日志
func Debug() *zerolog.Event {
	return Logger.Debug()
}

// Info 输出信息日志
func Info() *zerolog.Event {
	return Logger.Info()
}

// Warn 输出警告日志
func Warn() *zerolog.Event {
	return Logger.Warn()
}

// Error 输出错误日志
func Error() *zerolog.Event {
	return Logger.Error()
}

// Fatal 输出致命错误日志并退出
func Fatal() *zerolog.Event {
	return Logger.Fatal()
}

// Progress 输出进度信息
func Progress(current, total int, message string) {
	if total > 0 {
		percentage := float64(current) / float64(total) * 100
		Info().
			Int("current", current).
			Int("total", total).
			Float64("percentage", percentage).
			Msg(message)
	} else {
		Info().
			Int("current", current).
			Msg(message)
	}
}
