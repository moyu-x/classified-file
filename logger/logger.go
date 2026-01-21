package logger

import (
	"io"
	"os"
	"strings"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var Logger *zerolog.Logger

// Init 初始化 zerolog 日志
// level: 日志级别 ("debug", "info", "warn", "error")
// file: 日志文件路径，为空时仅输出到控制台
func Init(level string, file string) error {
	// 解析日志级别
	var logLevel zerolog.Level
	switch strings.ToLower(level) {
	case "debug":
		logLevel = zerolog.DebugLevel
	case "info":
		logLevel = zerolog.InfoLevel
	case "warn":
		logLevel = zerolog.WarnLevel
	case "error":
		logLevel = zerolog.ErrorLevel
	default:
		logLevel = zerolog.InfoLevel
	}

	// 配置输出
	var output io.Writer = os.Stdout

	if file != "" {
		// 如果指定了文件，同时输出到文件和控制台
		fileWriter, err := os.OpenFile(file, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		output = io.MultiWriter(os.Stdout, fileWriter)
	}

	// 设置全局 logger
	logger := log.Output(output).With().Timestamp().Logger().Level(logLevel)

	// 设置为控制台友好的格式
	logger = logger.Output(zerolog.ConsoleWriter{Out: output, TimeFormat: "2006-01-02 15:04:05"})

	Logger = &logger
	return nil
}

// Get 返回全局 logger 实例
// 如果 logger 未初始化，返回一个默认的 logger（输出到 /dev/null）
func Get() *zerolog.Logger {
	if Logger == nil {
		logger := zerolog.New(io.Discard)
		Logger = &logger
	}
	return Logger
}
