// SPDX-License-Identifier: MPL-2.0

package pdf

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsPrivateOrReservedIP(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		expected bool
	}{
		// Should be blocked (private/reserved)
		{"loopback IPv4", "127.0.0.1", true},
		{"loopback IPv4 alt", "127.0.0.2", true},
		{"loopback IPv6", "::1", true},
		{"private 10.x", "10.0.0.1", true},
		{"private 10.x alt", "10.255.255.255", true},
		{"private 172.16.x", "172.16.0.1", true},
		{"private 172.31.x", "172.31.255.255", true},
		{"private 192.168.x", "192.168.1.1", true},
		{"link-local IPv4 (metadata)", "169.254.169.254", true},
		{"link-local IPv4", "169.254.1.1", true},
		{"link-local IPv6", "fe80::1", true},
		{"unspecified IPv4", "0.0.0.0", true},
		{"unspecified IPv6", "::", true},

		// Should be allowed (public)
		{"public IPv4", "8.8.8.8", false},
		{"public IPv4 alt", "1.1.1.1", false},
		{"public IPv4 example", "93.184.216.34", false},
		{"public IPv6", "2001:4860:4860::8888", false},
		// Edge cases - 172.15.x and 172.32.x are public
		{"public 172.15.x", "172.15.255.255", false},
		{"public 172.32.x", "172.32.0.1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := net.ParseIP(tt.ip)
			assert.NotNil(t, ip, "failed to parse IP: %s", tt.ip)
			result := isPrivateOrReservedIP(ip)
			assert.Equal(t, tt.expected, result, "isPrivateOrReservedIP(%s)", tt.ip)
		})
	}

	// Test nil IP
	t.Run("nil IP", func(t *testing.T) {
		assert.True(t, isPrivateOrReservedIP(nil))
	})
}

func TestValidateImageURLHost(t *testing.T) {
	tests := []struct {
		name      string
		url       string
		wantError bool
	}{
		// Should be blocked
		{"localhost", "http://localhost/image.png", true},
		{"loopback IP", "http://127.0.0.1/image.png", true},
		{"loopback IP with port", "http://127.0.0.1:8080/image.png", true},
		{"AWS metadata", "http://169.254.169.254/latest/meta-data/", true},
		{"private 10.x", "http://10.0.0.1/internal.png", true},
		{"private 172.16.x", "http://172.16.0.1/internal.png", true},
		{"private 192.168.x", "http://192.168.1.1/internal.png", true},
		{"IPv6 loopback", "http://[::1]/image.png", true},
		{"missing hostname", "http:///path", true},

		// Should be allowed
		{"public domain", "https://example.com/image.png", false},
		{"public IP", "http://93.184.216.34/image.png", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateImageURLHost(tt.url)
			if tt.wantError {
				assert.Error(t, err, "expected error for URL: %s", tt.url)
			} else {
				assert.NoError(t, err, "unexpected error for URL: %s", tt.url)
			}
		})
	}
}

func TestDownloadImageAsBase64_DataURL(t *testing.T) {
	dataURL := "data:image/png;base64,iVBORw0KGgo="
	result, err := downloadImageAsBase64(dataURL)
	assert.NoError(t, err)
	assert.Equal(t, dataURL, result)
}

func TestDownloadImageAsBase64_InvalidScheme(t *testing.T) {
	_, err := downloadImageAsBase64("file:///etc/passwd")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported image URL scheme")
}

func TestDownloadImageAsBase64_PrivateIP(t *testing.T) {
	_, err := downloadImageAsBase64("http://127.0.0.1/image.png")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "private or reserved IP")
}

func TestDownloadImageAsBase64_MetadataEndpoint(t *testing.T) {
	_, err := downloadImageAsBase64("http://169.254.169.254/latest/meta-data/")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "private or reserved IP")
}
