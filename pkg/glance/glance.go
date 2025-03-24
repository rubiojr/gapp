package glance

import (
	"context"
	"fmt"
	"io"
	"log"

	g "github.com/rubiojr/gapp/internal/glance"
)

type Options struct {
	serverPort uint16
	serverHost string
	log        *log.Logger
}

func WithServerPort(port uint16) Option {
	return func(opts *Options) {
		opts.serverPort = port
	}
}

func WithLogger(log *log.Logger) Option {
	return func(opts *Options) {
		opts.log = log
	}
}

func WithHost(host string) Option {
	return func(opts *Options) {
		opts.serverHost = host
	}
}

type Option func(*Options)

func DefaultOptions() *Options {
	return &Options{
		serverPort: 65500,
		serverHost: "127.0.1.1",
		log:        log.New(io.Discard, "", 0),
	}
}

func ServeApp(ctx context.Context, cfg []byte, opts ...Option) error {
	log.SetOutput(io.Discard)

	config, err := g.NewConfigFromYAML(cfg)
	if err != nil {
		return fmt.Errorf("failed to parse config file: %v", err)
	}

	gopts := DefaultOptions()
	for _, opt := range opts {
		opt(gopts)
	}
	config.Server.Port = gopts.serverPort
	config.Server.Host = gopts.serverHost
	if gopts.log != nil {
		log.SetOutput(gopts.log.Writer())
	}

	app, err := g.NewApplication(config)
	if err != nil {
		return fmt.Errorf("failed to create application: %v", err)
	}

	var startServer func() error
	var stopServer func() error
	startServer, stopServer = app.Server()

	go func() {
		if err := startServer(); err != nil {
			log.Printf("server routine stopped: %v", err)
		}

		select {
		case <-ctx.Done():
			stopServer()
		}
	}()

	return nil
}
