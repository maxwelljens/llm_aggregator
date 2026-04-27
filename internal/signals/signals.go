// Package signals provides cross-platform signal handling for graceful shutdown.
package signals

import (
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
)

// Signals watched for graceful shutdown.
var (
	SigInt  = os.Signal(syscall.SIGINT)
	SigTerm = os.Signal(syscall.SIGTERM)
	SigHup  = os.Signal(syscall.SIGHUP)
)

// SignalHandler coordinates signal watching and exit requests.
type SignalHandler struct {
	notify  chan os.Signal
	done    chan struct{}
	exiting atomic.Bool
}

// New creates a new SignalHandler.
func New() *SignalHandler {
	return &SignalHandler{
		notify: make(chan os.Signal, 1),
		done:   make(chan struct{}),
	}
}

// Watch starts the signal listener goroutine and returns immediately.
// The goroutine exits cleanly when Stop() is called.
func (sh *SignalHandler) Watch() {
	signal.Notify(sh.notify, SigInt, SigTerm, SigHup)
	go func() {
		select {
		case <-sh.notify:
			sh.exiting.Store(true)
		case <-sh.done:
			return
		}
	}()
}

// IsExiting returns true if an exit has been requested (by signal).
func (sh *SignalHandler) IsExiting() bool {
	return sh.exiting.Load()
}

// Stop stops signal watching. The Watch() goroutine exits immediately.
// Safe to call from any goroutine, including signal handlers.
func (sh *SignalHandler) Stop() {
	signal.Stop(sh.notify)
	select {
	case sh.done <- struct{}{}:
	default:
	}
}
