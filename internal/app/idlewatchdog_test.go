package app_test

import (
	"errors"
	"io"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ozskywalker/ntfy-to-slack/internal/app"
)

// blockingReader blocks on Read until unblock is closed, then returns the
// given data (or error). It simulates a stream that has gone silent.
type blockingReader struct {
	unblock chan struct{}
	data    []byte
	err     error
}

func (r *blockingReader) Read(p []byte) (int, error) {
	<-r.unblock
	if r.err != nil {
		return 0, r.err
	}
	n := copy(p, r.data)
	return n, nil
}

func TestIdleWatchdogReader_TimeoutCancelsBlockedRead(t *testing.T) {
	underlying := &blockingReader{unblock: make(chan struct{})}
	defer close(underlying.unblock) // let the blocked Read return so the goroutine doesn't leak

	var canceled atomic.Bool
	cancel := func() { canceled.Store(true) }

	w := app.NewIdleWatchdogReader(underlying, 50*time.Millisecond, cancel)
	defer w.Stop()

	done := make(chan struct{})
	go func() {
		_, _ = w.Read(make([]byte, 1))
		close(done)
	}()

	select {
	case <-done:
		t.Fatal("Read returned before the underlying reader produced anything")
	case <-time.After(200 * time.Millisecond):
	}

	if !canceled.Load() {
		t.Error("expected cancel to have been called after the idle timeout")
	}
	if !w.TimedOut() {
		t.Error("expected TimedOut() to report true after the idle timeout fired")
	}
}

func TestIdleWatchdogReader_DataResetsTheTimer(t *testing.T) {
	underlying := &blockingReader{unblock: make(chan struct{}, 1), data: []byte("x")}

	var canceled atomic.Bool
	cancel := func() { canceled.Store(true) }

	const timeout = 80 * time.Millisecond
	w := app.NewIdleWatchdogReader(underlying, timeout, cancel)
	defer w.Stop()

	// Feed data faster than the timeout, several times in a row -- the
	// watchdog should never fire as long as data keeps arriving.
	for i := 0; i < 5; i++ {
		underlying.unblock <- struct{}{}
		if _, err := w.Read(make([]byte, 1)); err != nil {
			t.Fatalf("Read() error = %v", err)
		}
		time.Sleep(timeout / 2)
	}

	if canceled.Load() {
		t.Error("cancel should not have been called while data kept arriving within the timeout")
	}
	if w.TimedOut() {
		t.Error("TimedOut() should be false when the stream never actually went idle")
	}
}

func TestIdleWatchdogReader_OnReadCalledOnEverySuccessfulRead(t *testing.T) {
	underlying := &blockingReader{unblock: make(chan struct{}, 3), data: []byte("x")}

	w := app.NewIdleWatchdogReader(underlying, time.Second, func() {})
	defer w.Stop()

	var onReadCalls int
	w.OnRead = func() { onReadCalls++ }

	for i := 0; i < 3; i++ {
		underlying.unblock <- struct{}{}
		if _, err := w.Read(make([]byte, 1)); err != nil {
			t.Fatalf("Read() error = %v", err)
		}
	}

	if onReadCalls != 3 {
		t.Errorf("OnRead called %d times, want 3", onReadCalls)
	}
}

func TestIdleWatchdogReader_OnReadNotCalledOnError(t *testing.T) {
	underlying := &blockingReader{unblock: make(chan struct{}, 1), err: errors.New("boom")}
	underlying.unblock <- struct{}{}

	w := app.NewIdleWatchdogReader(underlying, time.Second, func() {})
	defer w.Stop()

	var onReadCalls int
	w.OnRead = func() { onReadCalls++ }

	if _, err := w.Read(make([]byte, 1)); err == nil {
		t.Fatal("expected the underlying error to propagate")
	}

	if onReadCalls != 0 {
		t.Errorf("OnRead called %d times on a failed Read, want 0", onReadCalls)
	}
}

func TestIdleWatchdogReader_StopDisarmsTheTimer(t *testing.T) {
	underlying := &blockingReader{unblock: make(chan struct{})}
	defer close(underlying.unblock)

	var canceled atomic.Bool
	cancel := func() { canceled.Store(true) }

	w := app.NewIdleWatchdogReader(underlying, 30*time.Millisecond, cancel)
	w.Stop()

	time.Sleep(100 * time.Millisecond)

	if canceled.Load() {
		t.Error("cancel should not be called after Stop()")
	}
}

func TestIdleWatchdogReader_PropagatesUnderlyingError(t *testing.T) {
	wantErr := errors.New("connection reset")
	underlying := &blockingReader{unblock: make(chan struct{}, 1), err: wantErr}
	underlying.unblock <- struct{}{}

	w := app.NewIdleWatchdogReader(underlying, time.Second, func() {})
	defer w.Stop()

	_, err := w.Read(make([]byte, 1))
	if !errors.Is(err, wantErr) {
		t.Errorf("Read() error = %v, want %v", err, wantErr)
	}
}

var _ io.Reader = (*blockingReader)(nil)
