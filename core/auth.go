package core

import (
	"context"
	"crypto/subtle"
	"shaper/util"

	"github.com/nats-io/nats.go/jetstream"
)

func ValidLogin(app *App, ctx context.Context, token string) (bool, error) {
	return subtle.ConstantTimeCompare([]byte(token), []byte(app.LoginToken)) == 1, nil
}

func ResetJWTSecret(app *App, ctx context.Context) ([]byte, error) {
	secret := util.GenerateRandomString(64)
	b := []byte(secret)
	_, err := app.ConfigKV.Put(ctx, CONFIG_KEY_JWT_SECRET, b)
	return b, err
}

func loadJWTSecret(app *App) error {
	entry, err := app.ConfigKV.Get(context.Background(), CONFIG_KEY_JWT_SECRET)
	if err == jetstream.ErrKeyNotFound {
		secret, err := ResetJWTSecret(app, context.Background())
		if err != nil {
			return err
		}
		app.JWTSecret = secret
	} else if err != nil {
		return err
	} else {
		app.JWTSecret = entry.Value()
	}
	return nil
}
