package main

import (
	"time"

	"github.com/mirzakhany/sysd"
)

func main() {
	systemd := sysd.New()
	systemd.SetGraceFulShutdownTimeout(4 * time.Second)
	systemd.SetStatusCheckInterval(1 * time.Second)

	if err := systemd.Add(&appA{}); err != nil {
		panic(err)
	}

	appB := &appB{}
	if err := systemd.Add(appB); err != nil {
		panic(err)
	}
	if err := systemd.SetAppOnFailure(appB.Name(), sysd.OnFailureIgnore); err != nil {
		panic(err)
	}

	appC := &appC{}
	if err := systemd.Add(appC); err != nil {
		panic(err)
	}
	if err := systemd.SetAppOnFailure(appC.Name(), sysd.OnFailureRestart.Retry(4).RetryTimeout(2*time.Second)); err != nil {
		panic(err)
	}

	// listen for os exit signals
	ctx := sysd.ContextWithSignals()
	// start all apps
	if err := systemd.Start(ctx); err != nil {
		panic(err)
	}
}
