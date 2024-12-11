package comms

import (
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

func New() (Comms, error) {
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
		// TODO: DontListen as default. No NATS exposed. Only internally
		// DontListen:             true,
		// TODO: StoreDir
		StoreDir: "./jetstream",
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
	clientOpts := []nats.Option{}
	// TODO: Make inprocess optional. Allow connecting to remote NATS
	clientOpts = append(clientOpts, nats.InProcessServer(ns))
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
