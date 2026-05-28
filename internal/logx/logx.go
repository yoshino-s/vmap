package logx

import (
	"fmt"
	"io"
	"os"
	"strings"
)

type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

func ParseLevel(input string) (Level, error) {
	normalized := strings.ToLower(strings.TrimSpace(input))
	if normalized == "" {
		normalized = "info"
	}

	switch normalized {
	case "debug":
		return LevelDebug, nil
	case "info":
		return LevelInfo, nil
	case "warn", "warning":
		return LevelWarn, nil
	case "error":
		return LevelError, nil
	default:
		return LevelInfo, fmt.Errorf("invalid log level %q, expected one of: debug|info|warn|error", input)
	}
}

type Logger struct {
	level Level
	out   io.Writer
}

func New(level Level, out io.Writer) *Logger {
	if out == nil {
		out = os.Stdout
	}
	return &Logger{level: level, out: out}
}

func NewFromString(input string, out io.Writer) (*Logger, error) {
	level, err := ParseLevel(input)
	if err != nil {
		return nil, err
	}
	return New(level, out), nil
}

func (l *Logger) Level() Level {
	return l.level
}

func (l *Logger) Enabled(level Level) bool {
	return level >= l.level
}

func (l *Logger) Debugf(format string, args ...any) {
	l.logf(LevelDebug, format, args...)
}

func (l *Logger) Infof(format string, args ...any) {
	l.logf(LevelInfo, format, args...)
}

func (l *Logger) Warnf(format string, args ...any) {
	l.logf(LevelWarn, format, args...)
}

func (l *Logger) Errorf(format string, args ...any) {
	l.logf(LevelError, format, args...)
}

func (l *Logger) logf(level Level, format string, args ...any) {
	if !l.Enabled(level) {
		return
	}
	_, _ = fmt.Fprintf(l.out, "[%s] %s\n", levelLabel(level), fmt.Sprintf(format, args...))
}

func levelLabel(level Level) string {
	switch level {
	case LevelDebug:
		return "debug"
	case LevelInfo:
		return "info"
	case LevelWarn:
		return "warn"
	case LevelError:
		return "error"
	default:
		return "info"
	}
}
