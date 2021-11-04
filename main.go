package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.uber.org/multierr"
	"golang.org/x/sync/errgroup"
)

type session struct {
	client *Client

	ctx    context.Context
	cancel context.CancelFunc

	done chan struct{}
	errc chan error
}

func newSession() *session {
	ctx, cancel := context.WithCancel(context.Background())
	return &session{
		client: NewClient(),
		ctx:    ctx,
		cancel: cancel,
		done:   make(chan struct{}),
		errc:   make(chan error, 1),
	}
}

func (s *session) checkData(ctx context.Context) error {
	close(s.done)
	<-ctx.Done()
	return nil
}

func (s *session) Start(ctx context.Context) error {
	go func(ctx context.Context) {
		err := s.client.Run(ctx, s.checkData)
		if err != nil && ctx.Err() == nil {
			s.unexpectedShutdown(ctx, err)
		}
		fmt.Println("err in gofunc RUN", err)
		s.errc <- err
	}(s.ctx)

	select {
	case <-ctx.Done():
		fmt.Println("ctx done")
		s.cancel()
		err := <-s.errc
		return multierr.Append(ctx.Err(), err)
	case err := <-s.errc:
		fmt.Println("s.errc")
		s.cancel()
		return err
	case <-s.done:
		fmt.Println("done RUN func")
		// OK
	}
	return nil

}

func (s *session) Stop(ctx context.Context) error {
	s.cancel()
	err := <-s.errc
	fmt.Println("s.cancel is STOP")
	return err
}

func (s *session) unexpectedShutdown(ctx context.Context, runError error) {
	log.Fatalf("unexpected shutdown: %v", runError)
}

type Client struct {
	runLock chan struct{}
}

func NewClient() *Client {
	return &Client{
		runLock: make(chan struct{}, 1),
	}
}

func (c *Client) Run(ctx context.Context, fsomething func(ctx context.Context) error) error {
	// select {
	// case <-ctx.Done():
	// 	return nil
	// case c.runLock <- struct{}{}:
	// 	defer func() { <-c.runLock }()
	// }

	if err := c.init(ctx); err != nil {
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
		c.loop(ctx)
		return nil
	})
	err := grp.Wait()

	return err
}

func (c *Client) init(ctx context.Context) error {
	t := time.NewTimer(10 * time.Millisecond)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
	}
	log.Println("init ended")
	return nil
}

func (c *Client) loop(ctx context.Context) {
	const interval = 200 * time.Millisecond
	var t *time.Timer
	for ctx.Err() == nil {
		if t == nil {
			t = time.NewTimer(interval)
			defer t.Stop()
		} else {
			t.Reset(interval)
		}
		select {
		case <-ctx.Done():
			fmt.Println("in loop ctx done")
			return
		case <-t.C:
		}
		log.Println("tick")
	}
}

func main() {
	ctx := context.Background()
	s := newSession()
	if err := withTimeout(s.Start, ctx, 500*time.Millisecond); err != nil {
		log.Fatalf("start: %+v", err)
	}
	time.Sleep(2 * time.Second)
	if err := s.Stop(ctx); err != nil {
		log.Fatalf("stop: %+v", err)
	}
}

func withTimeout(f func(ctx context.Context) error, ctx context.Context, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return f(ctx)
}
