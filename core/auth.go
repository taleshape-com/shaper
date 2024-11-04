package core

import (
	"context"
	"crypto/subtle"
)

func ValidLogin(app *App, ctx context.Context, token string) (bool, error) {
	return subtle.ConstantTimeCompare([]byte(token), []byte(app.LoginToken)) == 1, nil
}
