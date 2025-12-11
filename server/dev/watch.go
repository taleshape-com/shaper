// SPDX-License-Identifier: MPL-2.0

package dev

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
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
	"github.com/nrednav/cuid2"
	"github.com/syncthing/notify"
)

const (
	DASHBOARD_SUFFIX = ".dashboard.sql"
	shaperIDPrefix   = "-- shaperid:"
)

type DashboardClient interface {
	CreateDashboard(ctx context.Context, name, content, folderPath string) (string, error)
	SaveDashboardQuery(ctx context.Context, dashboardID, content string) error
}

type WatchConfig struct {
	WatchDirPath string
	Client       DashboardClient
	BaseURL      string
}

type Dev struct {
	c              chan notify.EventInfo
	server         *http.Server
	port           int
	connections    map[string][]*websocketConn // dashboardID -> connections
	connMutex      sync.RWMutex
	dashboardFiles map[string]string // dashboardID -> file path
	filesMutex     sync.RWMutex
	client         DashboardClient
	baseURL        string
	throttleMutex  sync.Mutex
	lastEventTime  time.Time
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
	if cfg.BaseURL == "" {
		cfg.BaseURL = "http://localhost:5454"
	}

	// Create Dev instance with websocket support
	dev := Dev{
		connections:    make(map[string][]*websocketConn),
		dashboardFiles: make(map[string]string),
		client:         cfg.Client,
		baseURL:        strings.TrimSuffix(cfg.BaseURL, "/"),
	}

	// Start websocket server on random port
	port, server, err := dev.startWebSocketServer()
	if err != nil {
		return nil, fmt.Errorf("failed to start websocket server: %w", err)
	}
	dev.port = port
	dev.server = server

	fmt.Println("Watching directory:", cfg.WatchDirPath)

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

	fileCount, err := ensureShaperIDsForDir(absWatchDir)
	if err != nil {
		return nil, fmt.Errorf("failed ensuring shaper IDs for dashboards in %s: %w", absWatchDir, err)
	}

	pluralSuffix := ""
	if fileCount != 1 {
		pluralSuffix = "s"
	}
	fmt.Printf("Found %d dashboard%s in watch directory.\n", fileCount, pluralSuffix)
	fmt.Printf("Dev server listening at :%d\n", port)
	fmt.Println(`
Create or edit any file with the .dashboard.sql extension in the watched directory.
A live-preview automatically opens in your browser.
The filename before the .dashboard.sql extension is the dashboard name.
Create sub-directories to organize dashboards into folders.`)

	if err := notify.Watch(path.Join(absWatchDir, "..."), c, notify.Create, notify.Write); err != nil {
		return nil, err
	}

	go func() {
		for ei := range c {
			p := ei.Path()
			dev.throttleFileEvent(p, func() {
				dev.handleDashboardFile(absWatchDir, p)
			})
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

const eventThrottleWindow = 500 * time.Millisecond

// throttleFileEvent implements a simple global throttling: after an event is handled,
// all events for any file are ignored for the next 500ms. This handles editor formatting
// events and Git branch switches where many files change at once.
func (d *Dev) throttleFileEvent(filePath string, handler func()) {
	d.throttleMutex.Lock()
	now := time.Now()
	elapsed := now.Sub(d.lastEventTime)

	// If we're within the throttle window, ignore this event
	if !d.lastEventTime.IsZero() && elapsed < eventThrottleWindow {
		d.throttleMutex.Unlock()
		return
	}

	// Update last event time and unlock before handling
	d.lastEventTime = now
	d.throttleMutex.Unlock()

	handler()
}

func (d *Dev) handleDashboardFile(absWatchDir, p string) {
	if !strings.HasSuffix(p, DASHBOARD_SUFFIX) {
		return
	}

	// TODO: on windows need to convert \ to /
	fPath, found := strings.CutPrefix(path.Dir(p), absWatchDir)
	if !found {
		fmt.Printf("ERROR: Failed removing prefix '%s' from dir %s\n", absWatchDir, path.Dir(p))
		return
	}
	name, found := strings.CutSuffix(path.Base(p), DASHBOARD_SUFFIX)
	if !found {
		fmt.Printf("ERROR: Failed removing suffix '%s' from file name '%s'\n", DASHBOARD_SUFFIX, path.Base(p))
		return
	}

	ctx := context.Background()
	contentBytes, updated, shaperID, err := ensureShaperIDForFile(p)
	if err != nil {
		fmt.Printf("ERROR: Failed ensuring ID comment in file '%s': %s\n", p, err)
		return
	}

	if updated {
		fmt.Printf("Set id '%s' for file '%s'\n", shaperID, p)
	}

	content := string(contentBytes)

	// Check if we have an existing dashboard for this file
	d.filesMutex.RLock()
	existingDashboardID := ""
	for dashID, filePath := range d.dashboardFiles {
		if filePath == p {
			existingDashboardID = dashID
			break
		}
	}
	d.filesMutex.RUnlock()

	var dashboardID string
	if existingDashboardID != "" {
		// Update existing dashboard
		err = d.client.SaveDashboardQuery(ctx, existingDashboardID, content)
		if err != nil {
			// Check if the error indicates the dashboard has expired (key not found)
			errStr := err.Error()
			if strings.Contains(errStr, "key not found") || strings.Contains(errStr, "failed to get dashboard") {
				// Dashboard expired, recreate it
				fmt.Printf("Temporary dashboard for '%s' expired, recreating\n", p)

				// Remove the expired dashboard from tracking
				d.filesMutex.Lock()
				delete(d.dashboardFiles, existingDashboardID)
				d.filesMutex.Unlock()

				// Create new dashboard
				dashboardID, err = d.client.CreateDashboard(ctx, name, content, fPath+"/")
				if err != nil {
					fmt.Printf("ERROR: Failed recreating expired dashboard for '%s': %s\n", p, err)
					return
				}

				// Track this file with new dashboard ID
				d.filesMutex.Lock()
				d.dashboardFiles[dashboardID] = p
				d.filesMutex.Unlock()

				fmt.Printf("Recreated expired dashboard for '%s%s'\n", fPath, name)

				url := fmt.Sprintf("%s/dashboards/%s?dev=ws://localhost:%d/ws", d.baseURL, dashboardID, d.port)
				if err := OpenURL(url); err != nil {
					fmt.Printf("ERROR: Failed opening '%s' in browser: %s\n", url, err)
				}
				return
			}

			// Other error, log and return
			fmt.Printf("ERROR: Failed updating dashboard '%s': %s\n", p, err)
			return
		}
		dashboardID = existingDashboardID
		fmt.Printf("Updated %s%s%s\n", fPath+"/", name, DASHBOARD_SUFFIX)

		// Notify websocket clients
		notified := d.notifyClients(dashboardID)
		if !notified {
			url := fmt.Sprintf("%s/dashboards/%s?dev=ws://localhost:%d/ws", d.baseURL, dashboardID, d.port)
			if err := OpenURL(url); err != nil {
				fmt.Printf("ERROR: Failed opening '%s' in browser: %s\n", url, err)
			}
		}
		return
	}

	// Create new dashboard
	dashboardID, err = d.client.CreateDashboard(ctx, name, content, fPath+"/")
	if err != nil {
		fmt.Printf("ERROR: Failed creating dashboard for '%s': %s\n", p, err)
		return
	}

	// Track this file
	d.filesMutex.Lock()
	d.dashboardFiles[dashboardID] = p
	d.filesMutex.Unlock()

	fmt.Printf("Created new dashboard for '%s%s'\n", fPath, name)

	url := fmt.Sprintf("%s/dashboards/%s?dev=ws://localhost:%d/ws", d.baseURL, dashboardID, d.port)
	if err := OpenURL(url); err != nil {
		fmt.Printf("ERROR: Failed opening '%s' in browser: %s\b", url, err)
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
			fmt.Printf("ERROR: WebSocket server error: %s\n", err)
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
			fmt.Printf("ERROR: WebSocket upgrade failed: %s\n", err)
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

		fmt.Printf("WebSocket connection established for dashboard '%s'\n", dashboardID)

		// Handle connection cleanup when it closes
		go func() {
			defer func() {
				conn.Close()
				d.removeConnection(dashboardID, wsConn.id)
				fmt.Printf("WebSocket connection closed for dashboard '%s'\n", dashboardID)
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

func hasLeadingShaperIDComment(content string) bool {
	if !strings.HasPrefix(content, shaperIDPrefix) {
		return false
	}

	lineEnd := strings.IndexByte(content, '\n')
	firstLine := content
	if lineEnd != -1 {
		firstLine = content[:lineEnd]
	}

	id := strings.TrimPrefix(firstLine, shaperIDPrefix)
	if id == "" || strings.ContainsAny(id, " \t\r") {
		return false
	}

	return true
}

func prependShaperIDComment(id, content string) string {
	commentLine := fmt.Sprintf("%s%s\n", shaperIDPrefix, id)
	if content != "" {
		if content[0] != '\n' && content[0] != '\r' {
			commentLine += "\n"
		}
		commentLine += content
		return commentLine
	}
	return commentLine + "\n"
}

func ensureShaperIDForFile(filePath string) ([]byte, bool, string, error) {
	contentBytes, err := os.ReadFile(filePath)
	if err != nil {
		return nil, false, "", err
	}

	content := string(contentBytes)
	if hasLeadingShaperIDComment(content) || strings.TrimSpace(content) == "" {
		return contentBytes, false, "", nil
	}

	newID := cuid2.Generate()
	newContent := prependShaperIDComment(newID, content)

	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, false, "", err
	}

	if err := os.WriteFile(filePath, []byte(newContent), fileInfo.Mode()); err != nil {
		return nil, false, "", err
	}

	return []byte(newContent), true, newID, nil
}

func ensureShaperIDsForDir(dir string) (int, error) {
	var aggregated error
	var fileCount int

	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(d.Name(), DASHBOARD_SUFFIX) {
			return nil
		}

		fileCount++

		_, updated, shaperID, err := ensureShaperIDForFile(p)
		if err != nil {
			fmt.Printf("ERROR: Failed ensuring ID comment in file '%s': %s\n", p, err)
			aggregated = errors.Join(aggregated, fmt.Errorf("%s: %w", p, err))
			return nil
		}

		if updated {
			fmt.Printf("Set id '%s' for file '%s'\n", shaperID, p)
		}

		return nil
	})

	if err != nil {
		return fileCount, err
	}

	return fileCount, aggregated
}
