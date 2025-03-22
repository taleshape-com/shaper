package comms

import (
	"context"
	"crypto/subtle"
	"fmt"
	"log/slog"
	"shaper/core"
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
	App        *core.App
}

type AuthCheckFunc func(context.Context, string) (bool, error)

type ClientAuth struct {
	Token []byte
	App   *core.App
}

func (c ClientAuth) Check(auth server.ClientAuthentication) bool {
	opts := auth.GetOpts()

	// First check static token
	if subtle.ConstantTimeCompare([]byte(opts.Token), c.Token) == 1 {
		auth.RegisterUser(c.createUser("root", true))
		return true
	}

	valid, err := core.ValidateAPIKey(c.App, context.Background(), opts.Token)
	if err != nil {
		return false
	}
	if valid {
		keyId := core.GetAPIKeyID(opts.Token)
		auth.RegisterUser(c.createUser("key."+keyId, false))
		return true
	}

	return false
}

func (c ClientAuth) createUser(name string, root bool) *server.User {
	if root {
		return &server.User{
			Username: name,
			Permissions: &server.Permissions{
				Publish: &server.SubjectPermission{
					Allow: []string{">"},
				},
				Subscribe: &server.SubjectPermission{
					Allow: []string{">"},
				},
			},
		}
	}
	return &server.User{
		Username: name,
		Permissions: &server.Permissions{
			Publish: &server.SubjectPermission{
				Allow: []string{"shaper.ingest.>"},
			},
			// TODO: jetstream publish is done via request/reply so we need inbox permissions to get the ACK,
			//       but it's not the most secure that the client can listen to all replies.
			//       Unfortunately NATS doesn't seem to support this scenario yet.
			Subscribe: &server.SubjectPermission{
				Allow: []string{"_INBOX.>"},
			},
			Response: &server.ResponsePermission{},
		},
	}
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
			App:   config.App,
		},
	}
	// Configure authentication if token is provided
	if config.Token != "" {
		opts.Authorization = config.Token
	} else {
		if !config.DontListen {
			config.Logger.Warn(fmt.Sprintf("nats: No nats-token provided and NATS is listening. If running in production make sure %s:%d is not exposed, configure a nats-token or set nats-dont-listen", config.Host, config.Port))
		}
	}
	if config.DontListen {
		config.Logger.Info("nats: Not listening on any network interfaces")
	}
	// Configure JetStream directory if provided
	opts.StoreDir = config.JSDir
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
	ns.Start()
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
