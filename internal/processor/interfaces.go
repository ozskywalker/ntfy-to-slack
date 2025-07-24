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
