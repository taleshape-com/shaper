package dev

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/syncthing/notify"
)

const DASHBOARD_SUFFIX = ".dashboard.sql"
const TIMEOUT = 10 * time.Second

type DashboardClient interface {
	CreateDashboard(ctx context.Context, name, content, folderPath string) (string, error)
	SaveDashboardQuery(ctx context.Context, dashboardID, content string) error
}

type WatchConfig struct {
	WatchDirPath string
	Client       DashboardClient
	Logger       *slog.Logger
	Addr         string
}

type Dev struct {
	c              chan notify.EventInfo
	server         *http.Server
	port           int
	connections    map[string][]*websocketConn // dashboardID -> connections
	connMutex      sync.RWMutex
	dashboardFiles map[string]string // dashboardID -> file path
	filesMutex     sync.RWMutex
	logger         *slog.Logger
	client         DashboardClient
	addr           string
}

type websocketConn struct {
	conn net.Conn
	id   string
}

type reloadMessage struct {
	Type        string `json:"type"`
	DashboardID string `json:"dashboardId"`
}

func Watch(cfg WatchConfig) (*Dev, error) {
	if cfg.WatchDirPath == "" {
		return nil, fmt.Errorf("watch directory is required")
	}
	if cfg.Client == nil {
		return nil, fmt.Errorf("dashboard client is required")
	}
	if cfg.Addr == "" {
		cfg.Addr = "localhost:5454"
	}

	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}

	// Create Dev instance with websocket support
	dev := Dev{
		connections:    make(map[string][]*websocketConn),
		dashboardFiles: make(map[string]string),
		logger:         logger,
		client:         cfg.Client,
		addr:           cfg.Addr,
	}

	// Start websocket server on random port
	port, server, err := dev.startWebSocketServer()
	if err != nil {
		return nil, fmt.Errorf("failed to start websocket server: %w", err)
	}
	dev.port = port
	dev.server = server

	dev.logger.Info("Watching dashboard files in dev mode",
		slog.String("dir", cfg.WatchDirPath),
		slog.Int("websocket_port", port))

	// Make the channel buffered to ensure no event is dropped. Notify will drop
	// an event if the receiver is not able to keep up the sending pace.
	c := make(chan notify.EventInfo, 1)
	dev.c = c

	// Set up a watchpoint listening on events within current working directory.
	// Dispatch each create and remove events separately to c.
	absWatchDir, err := filepath.Abs(cfg.WatchDirPath)
	if err != nil {
		return nil, err
	}
	if err := notify.Watch(path.Join(absWatchDir, "..."), c, notify.Create, notify.Write); err != nil {
		return nil, err
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
				dev.logger.Error("Failed removing prefix from dir of watched file", slog.String("dir", path.Dir(p)), slog.String("absWatchDir", absWatchDir))
				continue
			}
			name, found := strings.CutSuffix(path.Base(p), DASHBOARD_SUFFIX)
			if !found {
				dev.logger.Error("Failed removing dashboard suffix from watched file name", slog.String("file", path.Base(p)))
				continue
			}

			ctx, cancel := context.WithTimeout(context.Background(), TIMEOUT)

			// Read file content
			contentBytes, err := os.ReadFile(p)
			if err != nil {
				dev.logger.Error("Failed reading watched dashboard file", slog.String("file", p), slog.Any("error", err))
				cancel()
				continue
			}

			// Check if we have an existing dashboard for this file
			dev.filesMutex.RLock()
			existingDashboardID := ""
			for dashID, filePath := range dev.dashboardFiles {
				if filePath == p {
					existingDashboardID = dashID
					break
				}
			}
			dev.filesMutex.RUnlock()

			var dashboardID string
			if existingDashboardID != "" {
				// Update existing dashboard
				err = dev.client.SaveDashboardQuery(ctx, existingDashboardID, string(contentBytes))
				if err != nil {
					dev.logger.Error("Failed updating existing dashboard from watched file", slog.String("file", p), slog.Any("error", err))
					cancel()
					continue
				}
				dashboardID = existingDashboardID
				dev.logger.Info("Updated existing dashboard from file",
					slog.String("name", name),
					slog.String("path", fPath+"/"),
					slog.String("dashboard_id", dashboardID))

				// Notify websocket clients
				notified := dev.notifyClients(dashboardID)
				if !notified {
					url := fmt.Sprintf("http://%s/dashboards/%s?dev=ws://localhost:%d/ws", dev.addr, dashboardID, dev.port)
					if err := OpenURL(url); err != nil {
						dev.logger.Error("Failed opening dashboard in browser", slog.String("url", url), slog.Any("error", err))
					}
				}
			} else {
				// Create new dashboard
				dashboardID, err = dev.client.CreateDashboard(ctx, name, string(contentBytes), fPath+"/")
				if err != nil {
					dev.logger.Error("Failed creating dashboard from watched file", slog.String("file", p), slog.Any("error", err))
					cancel()
					continue
				}

				// Track this file
				dev.filesMutex.Lock()
				dev.dashboardFiles[dashboardID] = p
				dev.filesMutex.Unlock()

				dev.logger.Info("Created new dashboard from file",
					slog.String("name", name),
					slog.String("path", fPath),
					slog.String("dashboard_id", dashboardID))

				url := fmt.Sprintf("http://%s/dashboards/%s?dev=ws://localhost:%d/ws", dev.addr, dashboardID, dev.port)
				if err := OpenURL(url); err != nil {
					dev.logger.Error("Failed opening dashboard in browser", slog.String("url", url), slog.Any("error", err))
				}
			}
			cancel()
		}
	}()

	return &dev, nil
}

func (d *Dev) Stop() {
	notify.Stop(d.c)
	if d.server != nil {
		d.server.Close()
	}
}

// startWebSocketServer starts a websocket server on a random port
func (d *Dev) startWebSocketServer() (int, *http.Server, error) {
	// Find a random available port
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		return 0, nil, err
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", d.handleWebSocket())

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			d.logger.Error("WebSocket server error", slog.Any("error", err))
		}
	}()

	return port, server, nil
}

// handleWebSocket handles websocket connections
func (d *Dev) handleWebSocket() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dashboardID := r.URL.Query().Get("dashboardId")
		if dashboardID == "" {
			http.Error(w, "dashboardId parameter required", http.StatusBadRequest)
			return
		}

		conn, _, _, err := ws.UpgradeHTTP(r, w)
		if err != nil {
			d.logger.Error("WebSocket upgrade failed", slog.Any("error", err))
			return
		}

		wsConn := &websocketConn{
			conn: conn,
			id:   fmt.Sprintf("%s-%d", dashboardID, time.Now().UnixNano()),
		}

		// Add connection to the map
		d.connMutex.Lock()
		if d.connections[dashboardID] == nil {
			d.connections[dashboardID] = make([]*websocketConn, 0)
		}
		d.connections[dashboardID] = append(d.connections[dashboardID], wsConn)
		d.connMutex.Unlock()

		d.logger.Info("WebSocket connection established",
			slog.String("dashboardId", dashboardID),
			slog.String("connId", wsConn.id))

		// Handle connection cleanup when it closes
		go func() {
			defer func() {
				conn.Close()
				d.removeConnection(dashboardID, wsConn.id)
				d.logger.Info("WebSocket connection closed",
					slog.String("dashboardId", dashboardID),
					slog.String("connId", wsConn.id))
			}()

			// Keep connection alive by reading messages (though we don't expect any)
			for {
				_, _, err := wsutil.ReadClientData(conn)
				if err != nil {
					return
				}
			}
		}()
	}
}

// removeConnection removes a websocket connection from tracking
func (d *Dev) removeConnection(dashboardID, connID string) {
	d.connMutex.Lock()
	defer d.connMutex.Unlock()

	connections := d.connections[dashboardID]
	for i, conn := range connections {
		if conn.id == connID {
			d.connections[dashboardID] = append(connections[:i], connections[i+1:]...)
			break
		}
	}

	// Clean up empty dashboard entries
	if len(d.connections[dashboardID]) == 0 {
		delete(d.connections, dashboardID)
	}
}

// notifyClients sends a reload message to all connected clients for a dashboard
func (d *Dev) notifyClients(dashboardID string) bool {
	d.connMutex.RLock()
	connections := d.connections[dashboardID]
	d.connMutex.RUnlock()

	if len(connections) == 0 {
		return false
	}

	message := reloadMessage{
		Type:        "reload",
		DashboardID: dashboardID,
	}

	messageBytes := fmt.Sprintf(`{"type":"%s","dashboardId":"%s"}`, message.Type, message.DashboardID)

	for _, conn := range connections {
		go func(c *websocketConn) {
			err := wsutil.WriteServerMessage(c.conn, ws.OpText, []byte(messageBytes))
			if err != nil {
				// Connection is likely closed, it will be cleaned up by the read goroutine
				return
			}
		}(conn)
	}
	return true
}
