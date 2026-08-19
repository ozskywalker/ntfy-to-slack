package app_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ozskywalker/ntfy-to-slack/internal/app"
	"github.com/ozskywalker/ntfy-to-slack/internal/testutil"
)

func TestApp_Healthy_TrueImmediatelyAfterNew(t *testing.T) {
	a := app.New(&testutil.MockConfigProvider{Domain: "ntfy.sh", Topic: "t", WebhookURL: "https://hooks.slack.com/x"}, "test")

	if !a.Healthy(time.Minute) {
		t.Error("a freshly constructed App should be healthy")
	}
}

func TestApp_HealthHandler_ReportsOKWhileFresh(t *testing.T) {
	a := app.New(&testutil.MockConfigProvider{Domain: "ntfy.sh", Topic: "t", WebhookURL: "https://hooks.slack.com/x"}, "test")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	a.HealthHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Body.String(); got != `{"status":"ok"}` {
		t.Errorf("body = %q, want %q", got, `{"status":"ok"}`)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
}

func TestApp_Healthy_FalseBeforeAnyActivityRecorded(t *testing.T) {
	// A zero-value App (never passed through New, so recordActivity was
	// never called) has no sign of forward progress yet.
	var a app.App

	if a.Healthy(time.Hour) {
		t.Error("a zero-value App should not report healthy: it has never recorded any activity")
	}
}

func TestApp_Healthy_FalseOnceStale(t *testing.T) {
	a := app.New(&testutil.MockConfigProvider{Domain: "ntfy.sh", Topic: "t", WebhookURL: "https://hooks.slack.com/x"}, "test")

	// maxAge of 0 means "activity must be from this instant or later",
	// which the construction-time recordActivity call already fails by the
	// time this line runs.
	if a.Healthy(0) {
		t.Error("Healthy(0) should be false: even a just-recorded activity is technically in the past")
	}
}

func TestApp_HealthHandler_ReportsUnhealthyWhenStale(t *testing.T) {
	// This exercises the handler's response shape for the unhealthy branch
	// directly, without waiting out the real 5-minute staleness window: a
	// zero-value App (see TestApp_Healthy_FalseBeforeAnyActivityRecorded)
	// is unhealthy for the same reason a genuinely stale one would be --
	// Healthy(maxAge) is false -- so it exercises the same code path.
	var a app.App

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	a.HealthHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
	if got := rec.Body.String(); got != `{"status":"unhealthy"}` {
		t.Errorf("body = %q, want %q", got, `{"status":"unhealthy"}`)
	}
}
