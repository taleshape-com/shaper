package comms

import (
	"os"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
)

// TODO: Move consts to config
const (
	CONNECT_TIMEOUT = 10 * time.Second
)

type Comms struct {
	Conn   *nats.Conn
	Server *server.Server
}

type Config struct {
	Host       string
	Port       int
	Token      string
	JSDir      string
	JSKey      string
	MaxStore   int64
	DontListen bool
}

func New(config Config) (Comms, error) {
	// TODO: auth
	// TODO: allow changing nats host+port
	// TODO: support TLS
	// TODO: configure NATS logging
	// TODO: NATS prometheus metrics
	// TODO: JetStreamKey for disk encryption
	// TODO: Allow configuring stream retention
	opts := &server.Options{
		JetStream:              true,
		DisableJetStreamBanner: true,
		Host:                   config.Host,
		Port:                   config.Port,
		DontListen:             config.DontListen,
	}
	// Configure authentication if token is provided
	if config.Token != "" {
		opts.Authorization = config.Token
	}
	// Configure JetStream directory if provided
	if config.JSDir != "" {
		opts.StoreDir = config.JSDir
	} else {
		tmpStoreDir, err := os.MkdirTemp("", "shaper-nats")
		if err != nil {
			return Comms{}, err
		}
		opts.StoreDir = tmpStoreDir
	}
	// Configure JetStream encryption if key is provided
	if config.JSKey != "" {
		opts.JetStreamKey = config.JSKey
	}
	// Configure stream retention if set
	if config.MaxStore > 0 {
		opts.JetStreamMaxStore = config.MaxStore
	}

	ns, err := server.NewServer(opts)
	if err != nil {
		return Comms{}, err
	}
	ns.ConfigureLogger()
	go ns.Start()
	if !ns.ReadyForConnections(CONNECT_TIMEOUT) {
		return Comms{}, err
	}
	clientOpts := []nats.Option{
		// TODO: Make inprocess optional. Allow connecting to remote NATS
		nats.InProcessServer(ns),
	}

	// Add authentication to client if token is set
	if config.Token != "" {
		clientOpts = append(clientOpts, nats.Token(config.Token))
	}

	nc, err := nats.Connect(ns.ClientURL(), clientOpts...)
	if err != nil {
		return Comms{}, err
	}
	return Comms{Conn: nc, Server: ns}, err
}

func (c Comms) Close() {
	c.Server.Shutdown()
	c.Server.WaitForShutdown()
}
