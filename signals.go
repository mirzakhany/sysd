package sysd

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

// ContextWithSignal returns a context with by default is listening to
// SIGHUP, SIGINT, SIGTERM, SIGQUIT os signals to cancel
func ContextWithSignal(sig ...os.Signal) context.Context {
	if len(sig) == 0 {
		sig = []os.Signal{syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT}
	}

	s := make(chan os.Signal, 1)
	signal.Notify(s, sig...)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-s
		cancel()
	}()
	return ctx
}

// ShutdownGracefully will listen for context cancellation and call callback function if provided
func ShutdownGracefully(ctx context.Context, callback func() error) error {
	<-ctx.Done()
	if callback != nil {
		return (callback)()
	}
	return nil
}
