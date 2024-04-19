package log

import (
	"log/slog"
	"os"

	runtime "github.com/banzaicloud/logrus-runtime-formatter"
	"github.com/sirupsen/logrus"
)

type Level string

const (
	LevelTrace Level = "trace"
	LevelDebug Level = "debug"
	LevelInfo  Level = "info"
	LevelWarn  Level = "warn"
	LevelError Level = "error"
)

var levelVar *slog.LevelVar

// InitLogger will initialize the default logger instance.
func InitLogger() {
	levelVar = &slog.LevelVar{}
	levelVar.Set(slog.LevelInfo)

	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: levelVar, AddSource: true}))

	slog.SetDefault(logger)
}

// SetLevel will set the logging level of the default logger at runtime.
func SetLevel(loglevel string) {
	switch Level(loglevel) {
	case LevelDebug:
		levelVar.Set(slog.LevelDebug)
	case LevelInfo, "":
		levelVar.Set(slog.LevelInfo)
	case LevelWarn:
		levelVar.Set(slog.LevelWarn)
	case LevelError:
		levelVar.Set(slog.LevelError)
	default:
		levelVar.Set(slog.LevelInfo)
		slog.Warn("Unknown log level, defaulting to info", "loglevel", loglevel)
	}
}

// NewLogrusLogger will generate a new logrus logger instance
func NewLogrusLogger(logLevel string) *logrus.Logger {
	logger := logrus.New()

	logger.SetOutput(os.Stdout)

	switch Level(logLevel) {
	case LevelDebug:
		logger.Level = logrus.DebugLevel
	case LevelTrace:
		logger.Level = logrus.TraceLevel
	case LevelInfo, "":
		logger.Level = logrus.InfoLevel
	case LevelWarn:
		logger.Level = logrus.WarnLevel
	case LevelError:
		logger.Level = logrus.ErrorLevel
	default:
		logger.Level = logrus.InfoLevel
		logger.WithField("logLevel", logLevel).Warn("Unknown log level, defaulting to info")
	}

	logger.Level = logrus.InfoLevel

	runtimeFormatter := &runtime.Formatter{
		ChildFormatter: &logrus.JSONFormatter{},
		File:           true,
		Line:           true,
		BaseNameOnly:   true,
	}

	logger.SetFormatter(runtimeFormatter)

	return logger
}
