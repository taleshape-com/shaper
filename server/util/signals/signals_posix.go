// SPDX-License-Identifier: MPL-2.0

//go:build !windows

package signals

import (
	"os"
	"os/signal"
	"syscall"
)

// ListenHUP listens for SIGHUP signals in a background goroutine and calls onHUP each time.
// It returns a stop function to stop listening.
func ListenHUP(onHUP func()) func() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGHUP)
	done := make(chan struct{})

	go func() {
		for {
			select {
			case <-c:
				onHUP()
			case <-done:
				return
			}
		}
	}()

	return func() {
		signal.Stop(c)
		close(done)
	}
}
