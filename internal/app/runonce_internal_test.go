package app

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/ozskywalker/ntfy-to-slack/internal/testutil"
)

// fakeNtfyClient implements ntfy.Client with a scripted Connect result, so
// runOnce's post-Connect success path (watchdog wiring, ProcessStream,
// LastSeenTracker handling) can be exercised without a real ntfy server.
type fakeNtfyClient struct {
	reader io.ReadCloser
	err    error
	// blockUntilCanceled, if set, makes Connect block until ctx is done and
	// then return ctx.Err(), simulating a connection attempt that's still
	// in flight when the caller's context is canceled.
	blockUntilCanceled bool
}

func (f *fakeNtfyClient) Connect(ctx context.Context, since string) (io.ReadCloser, error) {
	if f.blockUntilCanceled {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	return f.reader, f.err
}

// nopCloser adapts an io.Reader to io.ReadCloser for tests that don't care
// about Close.
type nopCloser struct{ io.Reader }

func (nopCloser) Close() error { return nil }

// fakeStreamProcessor implements processor.StreamProcessor, and optionally
// processor.LastSeenTracker when withTracker is set, so both branches of
// runOnce's "if tracker, ok := a.processor.(processor.LastSeenTracker)"
// check can be exercised.
type fakeStreamProcessor struct {
	err error
}

func (f *fakeStreamProcessor) ProcessStream(reader io.Reader) error {
	// Drain the reader so a real io.Reader behind it (e.g. strings.Reader)
	// behaves like a StreamProcessor that actually reads its input.
	io.ReadAll(reader)
	return f.err
}

type fakeStreamProcessorWithTracker struct {
	fakeStreamProcessor
	lastSeenID string
	seen       bool
}

func (f *fakeStreamProcessorWithTracker) LastSeenID() (string, bool) {
	return f.lastSeenID, f.seen
}

func TestRunOnce_SuccessUpdatesSinceFromTracker(t *testing.T) {
	a := &App{
		config:     &testutil.MockConfigProvider{Domain: "ntfy.sh", Topic: "test"},
		ntfyClient: &fakeNtfyClient{reader: nopCloser{strings.NewReader("data")}},
		processor:  &fakeStreamProcessorWithTracker{lastSeenID: "msg-123", seen: true},
	}

	if err := a.runOnce(context.Background()); err != nil {
		t.Fatalf("runOnce() error = %v, want nil", err)
	}
	if a.since != "msg-123" {
		t.Errorf("since = %q, want %q", a.since, "msg-123")
	}
}

func TestRunOnce_SuccessWithoutTrackerLeavesSinceUnchanged(t *testing.T) {
	a := &App{
		config:     &testutil.MockConfigProvider{Domain: "ntfy.sh", Topic: "test"},
		ntfyClient: &fakeNtfyClient{reader: nopCloser{strings.NewReader("data")}},
		processor:  &fakeStreamProcessor{},
		since:      "unchanged",
	}

	if err := a.runOnce(context.Background()); err != nil {
		t.Fatalf("runOnce() error = %v, want nil", err)
	}
	if a.since != "unchanged" {
		t.Errorf("since = %q, want unchanged since processor has no LastSeenTracker", a.since)
	}
}

func TestRunOnce_ProcessStreamErrorIsReturned(t *testing.T) {
	wantErr := errors.New("stream processing failed")
	a := &App{
		config:     &testutil.MockConfigProvider{Domain: "ntfy.sh", Topic: "test"},
		ntfyClient: &fakeNtfyClient{reader: nopCloser{strings.NewReader("data")}},
		processor:  &fakeStreamProcessor{err: wantErr},
	}

	err := a.runOnce(context.Background())
	if !errors.Is(err, wantErr) {
		t.Fatalf("runOnce() error = %v, want %v", err, wantErr)
	}
}

// TestRun_ReturnsImmediatelyWhenContextIsAlreadyDoneAfterRunOnce covers the
// branch in Run's loop that checks ctx.Err() right after runOnce returns --
// distinct from the select's own <-ctx.Done() case, which only fires while
// waiting out the reconnect delay. This exercises the case where ctx is
// canceled while a connection attempt is still in flight.
func TestRun_ReturnsImmediatelyWhenContextIsAlreadyDoneAfterRunOnce(t *testing.T) {
	a := &App{
		config:     &testutil.MockConfigProvider{Domain: "ntfy.sh", Topic: "test"},
		ntfyClient: &fakeNtfyClient{blockUntilCanceled: true},
		processor:  &fakeStreamProcessor{},
	}

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() { done <- a.Run(ctx) }()

	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("Run() returned %v, want context.Canceled", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Run() did not return promptly after context cancellation")
	}
}

// TestRun_CleanStreamCloseLogsRestartAndRetries covers Run's "connection
// closed, restarting" branch, reached when runOnce returns a nil error
// (the stream ended without an error, as opposed to a connection failure).
func TestRun_CleanStreamCloseLogsRestartAndRetries(t *testing.T) {
	a := &App{
		config:     &testutil.MockConfigProvider{Domain: "ntfy.sh", Topic: "test"},
		ntfyClient: &fakeNtfyClient{reader: nopCloser{strings.NewReader("data")}},
		processor:  &fakeStreamProcessor{},
	}

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() { done <- a.Run(ctx) }()

	// Give Run one clean pass through runOnce (returns nil immediately,
	// since Connect/ProcessStream are both instant here) so it logs
	// "connection closed, restarting" and enters the reconnect wait, then
	// cancel rather than waiting out the real 30s reconnectDelay.
	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("Run() returned %v, want context.Canceled", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Run() did not return promptly after context cancellation")
	}
}
