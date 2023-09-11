package sysd

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"
)

const (
	GracefulShutdownTimeout = 20 * time.Second
	StatusCheckInterval     = 5 * time.Second
)

type OnFailure string

const (
	OnFailureRestart OnFailure = "restart"
	OnFailureStop    OnFailure = "stop"
	OnFailureIgnore  OnFailure = "ignore"
)

// ErrAppAlreadyExists is returned when an app is added to the systemd service
// but an app with the same name already exists
var ErrAppAlreadyExists = errors.New("app already exists")

type App interface {
	// Start starts the app, should be blocking until the context is cancelled
	Start(ctx context.Context) error
	// Stop stops the app,
	Stop(ctx context.Context) error
	// Status returns the status of the app
	Status(ctx context.Context) error
	// Name returns the name of the app
	Name() string
}

type appItem struct {
	App
	name      string
	onFailure OnFailure
}

// Systemd is a struct that represents a systemd service
type Systemd struct {
	apps             map[string]appItem
	defaultOnFailure OnFailure

	logger *logger

	graceFullShutdownTimeout time.Duration
	statusCheckInterval      time.Duration
}

// NewSystemd returns a new Systemd struct
func NewSystemd() *Systemd {
	return &Systemd{
		graceFullShutdownTimeout: GracefulShutdownTimeout,
		statusCheckInterval:      StatusCheckInterval,

		defaultOnFailure: OnFailureRestart,
		logger:           &logger{l: log.Default()},
	}
}

// Add adds an app to the systemd service
func (s *Systemd) Add(app App) error {
	if s.apps == nil {
		s.apps = make(map[string]appItem)
	}
	if _, ok := s.apps[app.Name()]; ok {
		s.logger.Error("app %q is already exist in systemd stack", app.Name())
		return ErrAppAlreadyExists
	}
	s.apps[app.Name()] = appItem{
		App:       app,
		name:      app.Name(),
		onFailure: s.defaultOnFailure,
	}
	return nil
}

func (s *Systemd) SetLogger(l Logger) {
	s.logger = &logger{l: l}
}

// SetGraceFulShutdownTimeout sets the graceful shutdown timeout
func (s *Systemd) SetGraceFulShutdownTimeout(t time.Duration) {
	s.graceFullShutdownTimeout = t
}

// SetStatusCheckInterval sets the status check interval
func (s *Systemd) SetStatusCheckInterval(t time.Duration) {
	s.statusCheckInterval = t
}

// SetDefaultOnFailure sets the default on failure action
func (s *Systemd) SetDefaultOnFailure(onFailure OnFailure) {
	s.defaultOnFailure = onFailure
}

func (s *Systemd) SetAppOnFailure(appName string, onFailure OnFailure) {
	if app, ok := s.apps[appName]; ok {
		app.onFailure = onFailure
		s.apps[appName] = app
	}
}

// Start starts the systemd service, and all apps within.
// it will return an error if any of the apps fail to start
// or block until the context is cancelled
func (s *Systemd) Start(ctx context.Context) error {
	// Start apps in parallel
	errs := make(chan error, len(s.apps))
	wg := sync.WaitGroup{}
	for _, app := range s.apps {
		s.startApp(ctx, app, &wg, errs)
	}

	go s.watchForStatus(ctx, &wg, errs)

	// wait for all apps to start or context to be cancelled
	for {
		select {
		case <-ctx.Done():
			return s.stopAllApps(context.Background())
		case err := <-errs:
			if !errors.Is(err, context.Canceled) {
				return err
			}
		}
	}
}

func (s *Systemd) startApp(ctx context.Context, app appItem, wg *sync.WaitGroup, errs chan error) {
	wg.Add(1)
	go func(app appItem) {
		defer func() {
			wg.Done()
			if r := recover(); r != nil {
				if err, ok := r.(error); ok {
					errs <- err
				} else {
					errs <- fmt.Errorf("%v", r)
				}
			}
		}()
		s.logger.Info("Starting app: %q", app.Name())
		errs <- app.Start(ctx)
	}(app)
}

// Stop stops the systemd service, and all apps within.
// it will return an error if any of the apps fail to stop
func (s *Systemd) stopAllApps(ctx context.Context) error {
	s.logger.Info("Stopping apps, gracefully shutdown timeout: %d", s.graceFullShutdownTimeout)
	shutDownCtx, cancel := context.WithTimeout(ctx, s.graceFullShutdownTimeout)
	defer cancel()

	// shutdown all apps
	errs := make(chan error, len(s.apps))
	wg := sync.WaitGroup{}
	for _, app := range s.apps {
		wg.Add(1)
		go func(app App) {
			defer wg.Done()
			if err := app.Stop(shutDownCtx); err != nil {
				errs <- err
			}
		}(app)
	}

	select {
	case <-shutDownCtx.Done():
		return shutDownCtx.Err()
	case err := <-errs:
		return err
	}
}

func (s *Systemd) watchForStatus(ctx context.Context, wg *sync.WaitGroup, errs chan error) {
	ticker := time.NewTicker(s.statusCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			errs <- ctx.Err()
		case <-ticker.C:
			for _, app := range s.apps {
				if err := app.Status(ctx); err != nil {
					s.logger.Error("app %q status check failed: %v", app.Name(), err)
					switch app.onFailure {
					case OnFailureRestart:
						s.logger.Info("Restarting app %q", app.Name())
						s.startApp(ctx, app, wg, errs)
					case OnFailureStop:
						s.logger.Info("Stopping app %q", app.Name())
						if err := app.Stop(ctx); err != nil {
							errs <- err
						}
					case OnFailureIgnore:
						s.logger.Info("Ignoring app %q failure", app.Name())
					}
				}
			}
		}
	}
}
