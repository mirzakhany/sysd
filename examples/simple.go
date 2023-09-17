package main

import (
	"syscall"
	"time"

	"github.com/mirzakhany/sysd"
)

func main() {
	systemd := sysd.NewSystemd()
	systemd.SetGraceFulShutdownTimeout(4 * time.Second)
	systemd.SetStatusCheckInterval(1 * time.Second)

	if err := systemd.Add(&appA{}); err != nil {
		panic(err)
	}

	appB := &appB{}
	if err := systemd.Add(appB); err != nil {
		panic(err)
	}
	systemd.SetAppOnFailure(appB.Name(), sysd.OnFailureIgnore)

	appC := &appC{}
	if err := systemd.Add(appC); err != nil {
		panic(err)
	}
	systemd.SetAppOnFailure(appC.Name(), sysd.OnFailureRestart.Retry(4).RetryTimeout(2*time.Second))

	// listen for os exit signals
	ctx := sysd.ContextWithSignal(syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT)
	// start all apps
	if err := systemd.Start(ctx); err != nil {
		panic(err)
	}
}
