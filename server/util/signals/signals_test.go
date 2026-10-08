// SPDX-License-Identifier: MPL-2.0

//go:build !windows

package signals

import (
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestListenHUP(t *testing.T) {
	hupReceived := make(chan struct{}, 1)
	stop := ListenHUP(func() {
		select {
		case hupReceived <- struct{}{}:
		default:
		}
	})
	defer stop()

	// Send SIGHUP to self
	err := syscall.Kill(syscall.Getpid(), syscall.SIGHUP)
	assert.NoError(t, err)

	select {
	case <-hupReceived:
		// success
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for SIGHUP callback")
	}
}
