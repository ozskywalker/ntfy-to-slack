package app

import (
	"net/http"
	"time"
)

// healthStaleness bounds how long Healthy tolerates no sign of forward
// progress before considering the app hung. It must comfortably exceed
// idleTimeout (the longest a healthy connection can go quiet before the
// idle watchdog itself forces a reconnect) and reconnectDelay (the longest
// a single failed-connection retry cycle takes), so that an external ntfy
// outage -- which Run's own retry loop is already correctly cycling
// through -- never trips this. Only a genuine stall (deadlock, a stuck
// goroutine) that stops both retries and reads should ever make the
// endpoint report unhealthy.
const healthStaleness = 5 * time.Minute

// recordActivity marks the current time as the most recent sign the app is
// making forward progress: either actively retrying a failed connection on
// schedule (called once per Run loop iteration), or actively receiving data
// on an open one (wired into the idle watchdog's OnRead, called on every
// successful Read). Either one means the app is doing its job, even if
// ntfy itself is unreachable right now -- restarting the process wouldn't
// fix an outage it doesn't control, so the health endpoint only reports
// unhealthy when the app has stopped doing *both* of these, not when ntfy
// has.
func (a *App) recordActivity() {
	a.lastActivity.Store(time.Now().UnixNano())
}

// Healthy reports whether the app has shown a sign of forward progress (see
// recordActivity) within maxAge.
func (a *App) Healthy(maxAge time.Duration) bool {
	last := a.lastActivity.Load()
	if last == 0 {
		// New App, recordActivity hasn't run yet (New calls it immediately,
		// so this is only reachable if someone constructs an App by hand).
		return false
	}
	return time.Since(time.Unix(0, last)) <= maxAge
}

// HealthHandler returns an http.Handler for a liveness endpoint: it reports
// the app's own liveness (is it still making forward progress), not ntfy's
// reachability. Mount it wherever convenient (e.g. "/healthz"); the handler
// itself is method- and path-agnostic.
func (a *App) HealthHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if a.Healthy(healthStaleness) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"ok"}`))
			return
		}
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(`{"status":"unhealthy"}`))
	})
}
