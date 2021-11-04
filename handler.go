package main

import (
	"context"
	"fmt"
	"log"

	"go.uber.org/multierr"
)

// Start for RunSomeService func
func (session *Session) Start(ctx context.Context) error {
	go func(ctx context.Context) {
		err := session.client.RunSomeService(ctx, session.checkSomeData)
		if err != nil && ctx.Err() == nil {
			session.UnexpectedShutdown(ctx, err)
		}
		fmt.Println("err: RunSomeService", err)
		session.errc <- err
	}(session.ctx)

	select {
	case <-ctx.Done():
		fmt.Println("ctx done: RunSomeService func")
		session.cancel()
		err := <-session.errc
		return multierr.Append(ctx.Err(), err)
	case err := <-session.errc:
		fmt.Println("s.errc: RunSomeService func")
		session.cancel()
		return err
	case <-session.done:
		fmt.Println("session done: RunSomeService func")
		// OK
	}

	return nil

}

func (session *Session) Stop(ctx context.Context) error {
	session.cancel()
	err := <-session.errc
	fmt.Println("s.cancel: Stop func")
	return err
}

func (session *Session) UnexpectedShutdown(ctx context.Context, runError error) {
	log.Fatalf("unexpected shutdown: %v", runError)
}

func (session *Session) checkSomeData(ctx context.Context) error {
	close(session.done)
	<-ctx.Done()
	return nil
}
