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
	// GracefulShutdownTimeout is the default graceful shutdown timeout
	GracefulShutdownTimeout = 20 * time.Second
	// StatusCheckInterval is the default status check interval
	StatusCheckInterval = 5 * time.Second
)

var (
	// OnFailureRestart will restart the app if it fails
	OnFailureRestart *OnFailure = &OnFailure{name: "restart", retry: 3, retryTimeout: 5 * time.Second}
	// OnFailureIgnore will ignore the app failure
	OnFailureIgnore *OnFailure = &OnFailure{name: "ignore"}

	// ErrAppAlreadyExists is returned when an app is added to the systemd service
	// but an app with the same name already exists
	ErrAppAlreadyExists = errors.New("app already exists")

	// ErrAppNotExists is returned when an app is not found in the systemd service
	ErrAppNotExists = errors.New("app not exists")
)

// OnFailure is an enum that represents the action to take when an app fails
type OnFailure struct {
	name         string
	retry        int
	retryTimeout time.Duration
}

// Equal returns true if the OnFailure is equal to the target
func (o *OnFailure) Equal(target *OnFailure) bool {
	return o.name == target.name
}

// String returns the string representation of the OnFailure
func (o *OnFailure) String() string {
	return o.name
}

// Retry set OnFailure number of retries
func (o *OnFailure) Retry(retry int) *OnFailure {
	o.retry = retry
	return o
}

// RetryTimeout set OnFailure retry timeout
func (o *OnFailure) RetryTimeout(retryTimeout time.Duration) *OnFailure {
	o.retryTimeout = retryTimeout
	return o
}

// App is an interface that represents an app
type App interface {
	// Start starts the app, should be blocking until the context is cancelled
	// restored is true if the app is being restored after a failure
	Start(ctx context.Context, restored bool) error
	// Status returns the status of the app
	Status(ctx context.Context) error
	// Name returns the name of the app
	Name() string
}

type appItem struct {
	App
	name      string
	onFailure *OnFailure
	priority  int
}

// Systemd is a struct that represents a systemd service
type Systemd struct {
	apps             map[string]appItem
	defaultOnFailure *OnFailure

	logger *logger

	graceFullShutdownTimeout time.Duration
	statusCheckInterval      time.Duration
}

// New returns a new Systemd struct
func New() *Systemd {
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
		priority:  0,
	}
	return nil
}

// SetLogger sets the logger
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
func (s *Systemd) SetDefaultOnFailure(onFailure *OnFailure) {
	s.defaultOnFailure = onFailure
}

// SetAppOnFailure sets the on failure action for a specific app
func (s *Systemd) SetAppOnFailure(appName string, onFailure *OnFailure) error {
	if app, ok := s.apps[appName]; ok {
		app.onFailure = onFailure
		s.apps[appName] = app
	}

	return ErrAppNotExists
}

// SetAppPriority sets the priority for a specific app
func (s *Systemd) SetAppPriority(appName string, priority int) error {
	if app, ok := s.apps[appName]; ok {
		app.priority = priority
		s.apps[appName] = app
	}

	return ErrAppNotExists
}

// Start starts the systemd service, and all apps within.
// it will return an error if any of the apps fail to start
// or block until the context is cancelled
func (s *Systemd) Start(ctx context.Context) error {
	// Start apps in parallel
	errs := make(chan error, len(s.apps))
	wg := sync.WaitGroup{}

	apps := make([]appItem, 0, len(s.apps))
	for _, app := range s.apps {
		apps = append(apps, app)
	}

	// sort apps by priority
	sortByPriority(apps)

	for _, app := range apps {
		s.startApp(ctx, app, &wg, errs, false)
	}

	go s.watchForStatus(ctx, &wg, errs)

	// wait for all apps to start or context to be cancelled
	for {
		select {
		case <-ctx.Done():
			s.WaitForAppsStop(&wg) // wait for all apps to stop
			return nil
		case err := <-errs:
			if !errors.Is(err, context.Canceled) {
				return err
			}
		}
	}
}

func sortByPriority(apps []appItem) {
	for i := 0; i < len(apps); i++ {
		for j := i + 1; j < len(apps); j++ {
			if apps[i].priority > apps[j].priority {
				apps[i], apps[j] = apps[j], apps[i]
			}
		}
	}
}

func (s *Systemd) startApp(ctx context.Context, app appItem, wg *sync.WaitGroup, errs chan error, restored bool) {
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
		// start the app with retry and timeout if configured
		if err := startWithRetry(ctx, app, restored); err != nil {
			errs <- err
		}
	}(app)
}

func startWithRetry(ctx context.Context, app appItem, restored bool) error {
	var err error
	for i := 0; i < app.onFailure.retry; i++ {
		if err = app.Start(ctx, restored); err != nil {
			time.Sleep(app.onFailure.retryTimeout)
			continue
		}
		return nil
	}
	return err
}

// WaitForAppsStop waits for all apps to stop or context to be cancelled
func (s *Systemd) WaitForAppsStop(wg *sync.WaitGroup) {
	// wait for all apps to stop or context to be cancelled
	select {
	case <-time.After(s.graceFullShutdownTimeout):
		s.logger.Error("Shutdown timeout, forcefully stopping apps")
		return
	case <-waitForGroup(wg):
		s.logger.Info("All apps stopped")
		return
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
						s.startApp(ctx, app, wg, errs, true)
					case OnFailureIgnore:
						s.logger.Info("Ignoring app %q failure", app.Name())
						// remove app from apps list
						delete(s.apps, app.Name())
						// remove app from wait group
						wg.Add(-1)
					}
				}
			}
		}
	}
}

func waitForGroup(wg *sync.WaitGroup) <-chan struct{} {
	c := make(chan struct{})
	go func() {
		wg.Wait()
		close(c)
	}()
	return c
}
