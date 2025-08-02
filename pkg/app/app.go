package app

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/sync/errgroup"

	"emailchecker/pkg/log"
)

type App struct {
	ctx context.Context

	webserver runnable
	runnables []runnable
}

func New(ctx context.Context) *App {
	ans := App{
		ctx: ctx,
	}

	return &ans
}

func (a *App) Run() error {
	log.Debug(a.ctx, "starting")

	ctx, cancel := context.WithCancel(a.ctx)
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		select {
		case sig := <-sigChan:
			log.Info(gCtx, "received signal", "signal", sig.String())
			cancel()
			return nil
		case <-gCtx.Done():
			return gCtx.Err()
		}
	})

	for _, r := range a.runnables {
		g.Go(func() error {
			return r.Run(gCtx)
		})
	}

	if a.webserver != nil {
		g.Go(func() error {
			return a.webserver.Run(gCtx)
		})
	}

	if err := g.Wait(); err != nil {
		return err
	}

	return nil
}

func (a *App) AddWebserver(webserver runnable) *App {
	if webserver == nil {
		panic("webserver cannot be nil")
	}

	if a.webserver != nil {
		log.Warn(a.ctx, "webserver already set, replacing")
	}

	a.webserver = webserver

	return a
}

func (a *App) Exec(r runnable) *App {
	if r == nil {
		panic("runnable cannot be nil")
	}

	a.runnables = append(a.runnables, r)

	return a
}

type runnable interface {
	Run(ctx context.Context) error
}
