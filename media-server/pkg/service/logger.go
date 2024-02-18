package service

import (
	"log/slog"
	"os"

	"go.uber.org/fx"
)

type logger_Params struct {
	fx.In
}

var loggerWriter = os.Stdout

func logger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(loggerWriter, &slog.HandlerOptions{
		AddSource: false,
		Level:     slog.LevelDebug,
	}))
}

var LoggerModule = fx.Module("logger", fx.Provide(
	logger,
))
