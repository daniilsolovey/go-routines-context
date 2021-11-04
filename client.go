package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"golang.org/x/sync/errgroup"
)

type Session struct {
	client *Client

	ctx    context.Context
	cancel context.CancelFunc

	done chan struct{}
	errc chan error
}

type Client struct {
}

func NewClient() *Client {
	return &Client{}
}

func NewSession() *Session {
	ctx, cancel := context.WithCancel(context.Background())
	return &Session{
		client: NewClient(),
		ctx:    ctx,
		cancel: cancel,
		done:   make(chan struct{}),
		errc:   make(chan error, 1),
	}
}

func (client *Client) RunSomeService(ctx context.Context, fsomething func(ctx context.Context) error) error {
	if err := client.initSomeData(ctx); err != nil {
		return err
	}

	grp, ctx := errgroup.WithContext(ctx)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	grp.Go(func() error {
		defer cancel()
		err := fsomething(ctx)
		return err
	})
	grp.Go(func() error {
		client.createLoopForSomeService(ctx)
		return nil
	})
	err := grp.Wait()

	return err
}

func (client *Client) initSomeData(ctx context.Context) error {
	t := time.NewTimer(10 * time.Millisecond)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
	}

	log.Println("init finished")
	return nil
}

func (client *Client) createLoopForSomeService(ctx context.Context) {
	const interval = 200 * time.Millisecond
	var timer *time.Timer
	for ctx.Err() == nil {
		if timer == nil {
			timer = time.NewTimer(interval)
			defer timer.Stop()
		} else {
			timer.Reset(interval)
		}
		select {
		case <-ctx.Done():
			fmt.Println("ctx done: loop func")
			return
		case <-timer.C:
		}

		log.Println("tick")
	}
}

func withTimeout(f func(ctx context.Context) error, ctx context.Context, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return f(ctx)
}
