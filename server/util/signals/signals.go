// SPDX-License-Identifier: MPL-2.0

package signals

import (
	"context"
	"os"
	"os/signal"
	"time"
)

func HandleInterrupt(onInterrupt func(context.Context)) {
	// Wait for interrupt signal to gracefully shutdown the server with a timeout of 10 seconds.
	ctx, stopInterruptNotify := signal.NotifyContext(context.Background(), os.Interrupt)
	<-ctx.Done()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	onInterrupt(ctx)
	cancel()
	stopInterruptNotify()
}
