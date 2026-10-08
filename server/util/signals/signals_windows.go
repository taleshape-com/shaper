// SPDX-License-Identifier: MPL-2.0

//go:build windows

package signals

// ListenHUP is a no-op on Windows since SIGHUP is not supported.
func ListenHUP(onHUP func()) func() {
	return func() {}
}
