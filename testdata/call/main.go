package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var ErrTaskComplete = errors.New("task complete")
var ErrSignal = errors.New("signal")

func main() {
	if len(os.Args) != 2 {
		fmt.Printf("Usage: %s <text>", os.Args[0])

		os.Exit(1)
	}
	name := os.Args[1]

	envVar := os.Getenv("ENV_VAR")
	cwd, _ := os.Getwd()

	fmt.Printf("ENV_VAR: %s\n", envVar)
	fmt.Printf("cwd: %s\n", cwd)

	ctx, done := context.WithCancelCause(context.Background())

	go func(ctx context.Context, done context.CancelCauseFunc) {
		for i := 0; i < 20; i++ {
			if i%2 == 0 {
				fmt.Fprintf(os.Stdout, "%s: stdout %d\n", name, i)
			} else {
				fmt.Fprintf(os.Stderr, "%s: stderr %d\n", name, i)
			}
			time.Sleep(500 * time.Millisecond)
		}

		done(ErrTaskComplete)
	}(ctx, done)

	signalC := make(chan os.Signal, 1)
	signal.Notify(signalC, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-signalC
		done(ErrSignal)
	}()

	<-ctx.Done()

	switch context.Cause(ctx) {
	case ErrSignal:
		fmt.Println("Exit by signal")
		os.Exit(1)
	case ErrTaskComplete:
		os.Exit(0)
	}
}
