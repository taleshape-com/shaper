package comms

import (
	"crypto/subtle"
	"log/slog"
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
	Logger     *slog.Logger
	Host       string
	Port       int
	Token      string
	JSDir      string
	JSKey      string
	MaxStore   int64
	DontListen bool
}

type ClientAuth struct {
	Token []byte
}

func (c ClientAuth) Check(auth server.ClientAuthentication) bool {
	opts := auth.GetOpts()
	valid := subtle.ConstantTimeCompare([]byte(opts.Token), c.Token) == 1
	auth.RegisterUser(&server.User{
		Username: "natstoken",
		Permissions: &server.Permissions{
			Publish: &server.SubjectPermission{
				Allow: []string{">"},
			},
			Subscribe: &server.SubjectPermission{
				Allow: []string{">"},
			},
		},
	})
	return valid
}

func New(config Config) (Comms, error) {
	// TODO: support TLS
	// TODO: NATS prometheus metrics
	// TODO: allow setting jetstream domain
	opts := &server.Options{
		JetStream:              true,
		DisableJetStreamBanner: true,
		Host:                   config.Host,
		Port:                   config.Port,
		DontListen:             config.DontListen,
		// We handle signals separately
		NoSigs: true,
		CustomClientAuthentication: ClientAuth{
			Token: []byte(config.Token),
		},
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
	ns.SetLoggerV2(newNATSLogger(config.Logger), false, false, false)
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

	// TODO: set nats.Name() for connection once we use more than one connection
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
