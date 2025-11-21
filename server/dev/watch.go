package dev

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"shaper/server/core"
	"strings"
	"time"

	"github.com/syncthing/notify"
)

const DASHBOARD_SUFFIX = ".dashboard.sql"
const TIMEOUT = 10 * time.Second

type Dev struct {
	c chan notify.EventInfo
}

func Watch(
	app *core.App,
	watchDirPath string,
) (Dev, error) {
	if watchDirPath == "" {
		return Dev{}, nil
	}
	app.Logger.Info("Watching dashboard files in dev mode", slog.String("dir", watchDirPath))
	// Make the channel buffered to ensure no event is dropped. Notify will drop
	// an event if the receiver is not able to keep up the sending pace.
	c := make(chan notify.EventInfo, 1)
	// Set up a watchpoint listening on events within current working directory.
	// Dispatch each create and remove events separately to c.
	absWatchDir, err := filepath.Abs(watchDirPath)
	if err != nil {
		return Dev{}, err
	}
	if err := notify.Watch(path.Join(absWatchDir, "..."), c, notify.Create, notify.Write); err != nil {
		return Dev{}, err
	}

	go func() {
		for ei := range c {
			p := ei.Path()
			if !strings.HasSuffix(p, DASHBOARD_SUFFIX) {
				continue
			}
			// TODO: on windows need to convert \ to /
			fPath, found := strings.CutPrefix(path.Dir(p), absWatchDir)
			if !found {
				app.Logger.Error("Failed removing prefix from dir of watched file", slog.String("dir", path.Dir(p)), slog.String("absWatchDir", absWatchDir))
				continue
			}
			name, found := strings.CutSuffix(path.Base(p), DASHBOARD_SUFFIX)
			if !found {
				app.Logger.Error("Failed removing dashboard suffix from watched file name", slog.String("file", path.Base(p)))
				continue
			}
			// TODO: set actor when logged in
			actor := "dev-watcher"
			ctx, cancel := context.WithTimeout(core.ContextWithActor(context.Background(), core.ActorFromString(actor)), TIMEOUT)
			defer cancel()
			// Read file content
			contentBytes, err := os.ReadFile(p)
			if err != nil {
				app.Logger.Error("Failed reading watched dashboard file", slog.String("file", p), slog.Any("error", err))
				continue
			}
			dashboardID, err := core.CreateDashboard(app, ctx, name, string(contentBytes), fPath+"/", true)
			if err != nil {
				app.Logger.Error("Failed reloading dashboard from watched file", slog.String("file", p), slog.Any("error", err))
				continue
			}
			log.Println("Dev: reloaded dashboard", name, "at", fPath, "with id", dashboardID)
			// TODO: get port from config
			port := 5454
			url := fmt.Sprintf("http://localhost:%d/dashboards/%s", port, dashboardID)
			if err := OpenURL(url); err != nil {
				app.Logger.Error("Failed opening dashboard in browser", slog.String("url", url), slog.Any("error", err))
			}
		}
	}()

	return Dev{c}, nil
}

func (d Dev) Stop() {
	notify.Stop(d.c)
}
