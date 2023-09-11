package main

import (
	"context"
	"log"

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
		log.Println("appA received shutdown signal")
		return nil
	})
}

func (a *appA) Stop(ctx context.Context) error {
	//TODO implement me
	panic("implement me")
}

func (a *appA) Status(ctx context.Context) error {
	return nil
}

func (a *appA) Name() string {
	return "appA"
}

type appB struct {
}

func (a *appB) Stop(ctx context.Context) error {
	//TODO implement me
	panic("implement me")
}

func (a *appB) Start(ctx context.Context) error {
	log.Println("appB started")

	defer func() {
		log.Println("appB stopped")
	}()

	return sysd.ShutdownGracefully(ctx, func() error {
		log.Println("appB received shutdown signal")
		return nil
	})
}

func (a *appB) Status(ctx context.Context) error {
	return nil
}

func (a *appB) Name() string {
	return "appB"
}
