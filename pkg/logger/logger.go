package logger

import (
	"context"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc/metadata"
)

type Logger struct {
	*zap.Logger
}

func New(env string) *Logger {
	var config zap.Config

	if env == "prod" || env == "production" {
		config = zap.NewProductionConfig()
		config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	} else {
		config = zap.NewDevelopmentConfig()
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	logger, err := config.Build()
	if err != nil {
		os.Stderr.WriteString("Failed to initialize zap logger: " + err.Error())
		os.Exit(1)
	}

	return &Logger{logger}
}

func (l *Logger) WithContext(ctx context.Context) *zap.Logger {
	fields := []zap.Field{}

	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if traceIDs := md.Get("x-trace-id"); len(traceIDs) > 0 {
			fields = append(fields, zap.String("trace_id", traceIDs[0]))
		}
	}
	if traceID, ok := ctx.Value("trace_id").(string); ok {
		fields = append(fields, zap.String("trace_id", traceID))
	}

	if len(fields) > 0 {
		return l.Logger.With(fields...)
	}
	return l.Logger
}
