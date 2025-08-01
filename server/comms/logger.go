// SPDX-License-Identifier: MPL-2.0

package comms

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/nats-io/nats-server/v2/server"
)

const natsLogPrefix = "nats: "

// Implement NATS logger with slog for more consistent logging output
type natsLogger struct {
	logger *slog.Logger
}

func newNATSLogger(logger *slog.Logger) server.Logger {
	return &natsLogger{
		logger: logger,
	}
}

func (n *natsLogger) Noticef(format string, v ...any) {
	n.logger.Info(natsLogPrefix + fmt.Sprintf(format, v...))
}

func (n *natsLogger) Warnf(format string, v ...any) {
	n.logger.Warn(natsLogPrefix + fmt.Sprintf(format, v...))
}

func (n *natsLogger) Fatalf(format string, v ...any) {
	n.logger.Error(natsLogPrefix+fmt.Sprintf(format, v...), slog.String("level", "fatal"))
	os.Exit(1)
}

func (n *natsLogger) Errorf(format string, v ...any) {
	n.logger.Error(natsLogPrefix + fmt.Sprintf(format, v...))
}

func (n *natsLogger) Debugf(format string, v ...any) {
	n.logger.Debug(natsLogPrefix + fmt.Sprintf(format, v...))
}

func (n *natsLogger) Tracef(format string, v ...any) {
	n.logger.Debug(natsLogPrefix+fmt.Sprintf(format, v...), slog.String("level", "trace"))
}
