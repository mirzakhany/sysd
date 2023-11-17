package main

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/mirzakhany/sysd"
)

type appA struct {
}

func (a *appA) Start(ctx context.Context) error {
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

type appB struct {
	failCounter int
}

func (a *appB) Start(ctx context.Context) error {
	log.Println("appB started")

	// rest failure counter
	a.failCounter = 0

	defer func() {
		log.Println("appB stopped")
	}()

	return sysd.ShutdownGracefully(ctx, func() error {
		log.Println("appB received shutdown signal")
		return nil
	})
}

func (a *appB) Status(ctx context.Context) error {
	log.Println("appB status")
	a.failCounter++
	if a.failCounter > 3 {
		return errors.New("appB failed")
	}
	return nil
}

func (a *appB) Name() string {
	return "appB"
}

type appC struct {
	sucessCounter  int
	restoreCounter int
}

func (a *appC) Start(ctx context.Context) error {
	log.Println("appC started")

	if sysd.IsRestored(ctx) {
		a.restoreCounter++
	}

	if a.restoreCounter > 2 {
		// lets hard fail the app after 2 retries
		return errors.New("appC failed to restore")
	}

	defer func() {
		log.Println("appC stopped")
	}()

	return sysd.ShutdownGracefully(ctx, func() error {
		log.Println("appC received shutdown signal")
		return nil
	})
}

func (a *appC) Status(ctx context.Context) error {
	log.Println("appC status")
	a.sucessCounter++

	// fail until the app is restored
	if a.sucessCounter > 2 && a.restoreCounter <= 2 {
		return errors.New("appC failed")
	}
	return nil
}

func (a *appC) Name() string {
	return "appC"
}
