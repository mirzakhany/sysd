package httpd

import (
	"context"
	"net"
	"net/http"
	"strconv"

	"github.com/mirzakhany/sysd"
)

var _ sysd.App = &HTTPd{}

type HTTPd struct {
	Host string
	Port int

	handler http.Handler

	server *http.Server
}

func (h *HTTPd) New(Host string, Port int, handler http.Handler) *HTTPd {
	return &HTTPd{
		Host:    Host,
		Port:    Port,
		handler: handler,
	}
}

func (h *HTTPd) Start(ctx context.Context) error {
	srv := &http.Server{
		Addr:    net.JoinHostPort(h.Host, strconv.Itoa(h.Port)),
		Handler: h.handler,
	}

	if err := srv.ListenAndServe(); err != nil {
		return err
	}

	h.server = srv

	return sysd.ShutdownGracefully(ctx, func() error {
		return srv.Shutdown(ctx)
	})
}

func (h *HTTPd) Status(ctx context.Context) error {
	// TODO: check if server is running and return error if not
	return nil
}

func (h *HTTPd) Name() string {
	return "httpd"
}
