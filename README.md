# Sysd

Sysd is an app Life cycle manager. it runs the apps and restarts them if they crash. 
It also provides a simple way to manage the apps.

## Installation
To install sysd, use `go get`:

```shell
    go get github.com/abhishekkr/sysd
```

Sample application:

```go
package main

import (
	"context"
	"log"
	"time"

	"github.com/mirzakhany/sysd"
)

type appA struct {
}

func (a *appA) Start(ctx context.Context, restored bool) error {
	log.Println("appA started")

	defer func() {
		log.Println("appA stopped")
	}()

	return sysd.ShutdownGracefully(ctx, func() error {
		time.Sleep(5 * time.Second)
		log.Println("appA received shutdown signal")
		return nil
	})
}

func (a *appA) Status(ctx context.Context) error {
	log.Println("appA status")
	return nil
}

func (a *appA) Name() string {
	return "appA"
}

func main() {
	systemd := sysd.New()
	systemd.SetGraceFulShutdownTimeout(4 * time.Second)
	systemd.SetStatusCheckInterval(1 * time.Second)

	a := &appA{}
	if err := systemd.Add(a); err != nil {
		panic(err)
	}

	// listen for os exit signals
	ctx := sysd.ContextWithSignals()
	// start all apps
	if err := systemd.Start(ctx); err != nil {
		panic(err)
	}
}
```


