package main

import (
	"context"
	"log"
	"time"
)

func main() {
	ctx := context.Background()
	session := NewSession()
	if err := withTimeout(session.Start, ctx, 500*time.Millisecond); err != nil {
		log.Fatalf("start: %+v", err)
	}

	time.Sleep(2 * time.Second)
	if err := session.Stop(ctx); err != nil {
		log.Fatalf("session stopped: %+v", err)
	}
}
