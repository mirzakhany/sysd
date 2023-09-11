package main

import (
	"syscall"

	"github.com/mirzakhany/sysd"
)

func main() {
	systemd := sysd.NewSystemd()
	if err := systemd.Add(&appA{}); err != nil {
		panic(err)
	}
	if err := systemd.Add(&appB{}); err != nil {
		panic(err)
	}

	// listen for os exit signals
	ctx := sysd.ContextWithSignal(syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT)
	// start all apps
	if err := systemd.Start(ctx); err != nil {
		panic(err)
	}
}
