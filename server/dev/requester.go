package dev

import (
	"context"
	"net/http"
)

type appsRequester interface {
	DoRequest(ctx context.Context, method, path string, body []byte) (*http.Response, error)
}
