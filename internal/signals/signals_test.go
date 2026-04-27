package signals

import (
	"testing"
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
