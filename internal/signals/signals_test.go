package signals

import (
	"os"
	"syscall"
	"testing"
	"time"
)

func TestSignalHandler_Watch(t *testing.T) {
	sh := New()
	sh.Watch()
	sh.Stop()
}

func TestSignalHandler_IsExiting(t *testing.T) {
	sh := New()
	sh.Watch()
	sh.Stop()

	if sh.IsExiting() {
		t.Fatal("IsExiting() should be false after Stop()")
	}
}

func TestSignalHandler_Stop_Idempotent(t *testing.T) {
	sh := New()
	sh.Watch()
	sh.Stop()
	sh.Stop()
	sh.Stop()
}

func TestSignalHandler_Done(t *testing.T) {
	sh := New()
	sh.Watch()
	defer sh.Stop()

	proc, err := os.FindProcess(os.Getpid())
	if err != nil {
		t.Fatalf("FindProcess: %v", err)
	}
	if err := proc.Signal(syscall.SIGTERM); err != nil {
		t.Fatalf("Signal: %v", err)
	}

	select {
	case <-sh.Done():
	case <-time.After(2 * time.Second):
		t.Fatal("Done() not closed after signal")
	}

	if !sh.IsExiting() {
		t.Fatal("IsExiting() should be true after signal")
	}
}
