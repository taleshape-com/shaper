package comms

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/nats-io/nats-server/v2/server"
)

type natsLogger struct {
	logger *slog.Logger
}

func newNATSLogger(logger *slog.Logger) server.Logger {
	return &natsLogger{
		logger: logger,
	}
}

func (n *natsLogger) Noticef(format string, v ...any) {
	n.logger.Info(fmt.Sprintf(format, v...))
}

func (n *natsLogger) Warnf(format string, v ...any) {
	n.logger.Warn(fmt.Sprintf(format, v...))
}

func (n *natsLogger) Fatalf(format string, v ...any) {
	n.logger.Error(fmt.Sprintf(format, v...), slog.String("level", "fatal"))
	os.Exit(1)
}

func (n *natsLogger) Errorf(format string, v ...any) {
	n.logger.Error(fmt.Sprintf(format, v...))
}

func (n *natsLogger) Debugf(format string, v ...any) {
	n.logger.Debug(fmt.Sprintf(format, v...))
}

func (n *natsLogger) Tracef(format string, v ...any) {
	n.logger.Debug(fmt.Sprintf(format, v...), slog.String("level", "trace"))
}
