package logger

import (
	"io"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Logger struct {
	*zap.Logger
}

type Config struct {
	Level      string
	Production bool
	Output     io.Writer // For testing
	JsonFormat bool      // Use JSON format instead of console format
}

// New creates a new logger with the given configuration
func New(config Config) *Logger {
	var level zapcore.Level
	if err := level.UnmarshalText([]byte(config.Level)); err != nil {
		level = zapcore.InfoLevel
	}

	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "message",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	if config.Production {
		encoderConfig.EncodeLevel = zapcore.LowercaseLevelEncoder
		encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	}

	var encoder zapcore.Encoder
	if config.JsonFormat || config.Production {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	}

	var writeSyncer zapcore.WriteSyncer
	if config.Output != nil {
		writeSyncer = zapcore.AddSync(config.Output)
	} else {
		writeSyncer = zapcore.AddSync(os.Stdout)
	}

	core := zapcore.NewCore(encoder, writeSyncer, level)

	options := []zap.Option{
		zap.AddCaller(),
		zap.AddCallerSkip(1),
	}

	if !config.Production {
		options = append(options, zap.Development())
		options = append(options, zap.AddStacktrace(zapcore.ErrorLevel))
	}

	zapLogger := zap.New(core, options...)

	return &Logger{
		Logger: zapLogger,
	}
}

func (l *Logger) With(fields ...zapcore.Field) *Logger {
	return &Logger{
		Logger: l.Logger.With(fields...),
	}
}

func (l *Logger) WithField(key string, value interface{}) *Logger {
	return l.With(zap.Any(key, value))
}

func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	zapFields := make([]zapcore.Field, 0, len(fields))
	for k, v := range fields {
		zapFields = append(zapFields, zap.Any(k, v))
	}
	return l.With(zapFields...)
}

var (
	defaultLogger *Logger
)

func InitLogger(config Config) {
	defaultLogger = New(config)
}

func Default() *Logger {
	if defaultLogger == nil {
		defaultLogger = New(Config{
			Level:      "info",
			Production: false,
		})
	}
	return defaultLogger
}

func Debug(msg string, fields ...zapcore.Field) {
	Default().Debug(msg, fields...)
}

func Info(msg string, fields ...zapcore.Field) {
	Default().Info(msg, fields...)
}

func Warn(msg string, fields ...zapcore.Field) {
	Default().Warn(msg, fields...)
}

func Error(msg string, fields ...zapcore.Field) {
	Default().Error(msg, fields...)
}

func Fatal(msg string, fields ...zapcore.Field) {
	Default().Fatal(msg, fields...)
}
