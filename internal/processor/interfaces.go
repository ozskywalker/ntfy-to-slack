package processor

import (
	"io"
	"net/http"

	"github.com/ozskywalker/ntfy-to-slack/internal/config"
)

// HTTPClient interface for HTTP operations
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// MessageSender interface for sending messages to external services
type MessageSender interface {
	Send(message *config.SlackMessage) error
}

// StreamProcessor interface for processing ntfy message streams
type StreamProcessor interface {
	ProcessStream(reader io.Reader) error
}

// LastSeenTracker is an optional interface a StreamProcessor may implement
// to report the id of the most recent message it saw. A caller that gets
// back a StreamProcessor implementing this (via a type assertion, since it
// isn't part of StreamProcessor itself) can use the id to resume the ntfy
// stream via its "since" query parameter after a reconnect, instead of
// either redelivering or losing messages across the gap.
type LastSeenTracker interface {
	LastSeenID() (id string, ok bool)
}
