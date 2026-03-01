package logger

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
)

// Level 日志级别
type Level int

const (
	DebugLevel Level = iota
	InfoLevel
	WarnLevel
	ErrorLevel
	FatalLevel
)

func (l Level) String() string {
	switch l {
	case DebugLevel:
		return "DEBUG"
	case InfoLevel:
		return "INFO"
	case WarnLevel:
		return "WARN"
	case ErrorLevel:
		return "ERROR"
	case FatalLevel:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// Logger 日志记录器
type Logger struct {
	level      Level
	output     io.Writer
	colorful   bool
	timeFormat string
	mu         sync.Mutex
}

// 全局日志实例
var defaultLogger = NewLogger()

// NewLogger 创建新的日志记录器
func NewLogger() *Logger {
	return &Logger{
		level:      InfoLevel,
		output:     os.Stderr,
		colorful:   true,
		timeFormat: "15:04:05",
	}
}

// SetLevel 设置日志级别
func (l *Logger) SetLevel(level Level) {
	l.level = level
}

// SetOutput 设置输出目标
func (l *Logger) SetOutput(w io.Writer) {
	l.output = w
}

// DisableColor 禁用颜色
func (l *Logger) DisableColor() {
	l.colorful = false
}

// log 实际记录日志
func (l *Logger) log(level Level, format string, args ...interface{}) {
	if level < l.level {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	// 格式化消息
	msg := fmt.Sprintf(format, args...)

	// 构建日志行
	parts := []string{}

	// 时间
	if l.timeFormat != "" {
		parts = append(parts, time.Now().Format(l.timeFormat))
	}

	// 级别标签
	levelStr := l.formatLevel(level)
	parts = append(parts, levelStr)

	// 消息
	parts = append(parts, msg)

	// 输出
	line := strings.Join(parts, " ") + "\n"
	l.output.Write([]byte(line))

	// Fatal 级别退出程序
	if level == FatalLevel {
		os.Exit(1)
	}
}

// formatLevel 格式化级别标签
func (l *Logger) formatLevel(level Level) string {
	if !l.colorful {
		return fmt.Sprintf("[%s]", level.String())
	}

	switch level {
	case DebugLevel:
		return color.BlueString("[DEBUG]")
	case InfoLevel:
		return color.GreenString("[INFO]")
	case WarnLevel:
		return color.YellowString("[WARN]")
	case ErrorLevel:
		return color.RedString("[ERROR]")
	case FatalLevel:
		return color.HiRedString("[FATAL]")
	default:
		return fmt.Sprintf("[%s]", level.String())
	}
}

// Debug 调试日志
func (l *Logger) Debug(format string, args ...interface{}) {
	l.log(DebugLevel, format, args...)
}

// Info 信息日志
func (l *Logger) Info(format string, args ...interface{}) {
	l.log(InfoLevel, format, args...)
}

// Warn 警告日志
func (l *Logger) Warn(format string, args ...interface{}) {
	l.log(WarnLevel, format, args...)
}

// Error 错误日志
func (l *Logger) Error(format string, args ...interface{}) {
	l.log(ErrorLevel, format, args...)
}

// Fatal 致命错误日志
func (l *Logger) Fatal(format string, args ...interface{}) {
	l.log(FatalLevel, format, args...)
}

// ========== 包级函数 ==========

// SetLevel 设置全局日志级别
func SetLevel(level Level) {
	defaultLogger.SetLevel(level)
}

// SetDebug 启用调试模式
func SetDebug() {
	defaultLogger.SetLevel(DebugLevel)
}

// DisableColor 禁用颜色
func DisableColor() {
	defaultLogger.DisableColor()
}

// Debug 调试日志
func Debug(format string, args ...interface{}) {
	defaultLogger.Debug(format, args...)
}

// Info 信息日志
func Info(format string, args ...interface{}) {
	defaultLogger.Info(format, args...)
}

// Warn 警告日志
func Warn(format string, args ...interface{}) {
	defaultLogger.Warn(format, args...)
}

// Error 错误日志
func Error(format string, args ...interface{}) {
	defaultLogger.Error(format, args...)
}

// Fatal 致命错误日志
func Fatal(format string, args ...interface{}) {
	defaultLogger.Fatal(format, args...)
}
