package app

import (
	"io"
	"sync/atomic"
	"time"
)

// IdleWatchdogReader wraps a reader with an idle timeout. ntfy sends a
// keepalive event roughly every 45 seconds on an open connection, so
// bytes should always arrive well within timeout on a healthy stream; a
// network partition or other silent failure that doesn't produce a TCP
// RST/FIN, though, leaves a blocking Read() waiting forever with no error
// to react to. If no Read succeeds within timeout of the last one (or of
// construction, for the first), the watchdog calls cancel so the reader's
// underlying request is aborted and the blocked Read returns an error
// instead of hanging indefinitely.
type IdleWatchdogReader struct {
	r        io.Reader
	timeout  time.Duration
	timer    *time.Timer
	timedOut atomic.Bool
}

// NewIdleWatchdogReader creates a watchdog around r. cancel is called at
// most once, the first time the stream goes timeout without producing any
// data.
func NewIdleWatchdogReader(r io.Reader, timeout time.Duration, cancel func()) *IdleWatchdogReader {
	w := &IdleWatchdogReader{r: r, timeout: timeout}
	w.timer = time.AfterFunc(timeout, func() {
		w.timedOut.Store(true)
		cancel()
	})
	return w
}

// Read implements io.Reader, resetting the idle timer on every successful
// read.
func (w *IdleWatchdogReader) Read(p []byte) (int, error) {
	n, err := w.r.Read(p)
	if n > 0 {
		w.timer.Reset(w.timeout)
	}
	return n, err
}

// Stop disarms the watchdog. Call it once the stream is done with, whether
// that's because it ended on its own or the caller is discarding it for
// another reason, so a late timer firing can't call cancel after the fact.
func (w *IdleWatchdogReader) Stop() {
	w.timer.Stop()
}

// TimedOut reports whether the watchdog is the reason its cancel was
// called, i.e. whether the stream actually went idle for longer than
// timeout (as opposed to being canceled for some unrelated reason, such as
// the caller's own context being canceled).
func (w *IdleWatchdogReader) TimedOut() bool {
	return w.timedOut.Load()
}
