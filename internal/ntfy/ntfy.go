package ntfy

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/ozskywalker/ntfy-to-slack/internal/config"
)

// Client interface for ntfy operations
type Client interface {
	Connect() (io.ReadCloser, error)
}

// HTTPClient implements Client interface
type HTTPClient struct {
	domain string
	topic  string
	auth   string
	client config.HTTPClient
}

// NewClient creates a new ntfy client
func NewClient(domain, topic, auth string, client config.HTTPClient) *HTTPClient {
	if client == nil {
		client = &http.Client{}
	}
	return &HTTPClient{
		domain: domain,
		topic:  topic,
		auth:   auth,
		client: client,
	}
}

// Connect implements Client interface
func (n *HTTPClient) Connect() (io.ReadCloser, error) {
	// Validate inputs
	domain, err := config.ValidateDomain(n.domain)
	if err != nil {
		return nil, err
	}

	topic, err := config.ValidateTopic(n.topic)
	if err != nil {
		return nil, err
	}

	baseURL := fmt.Sprintf("https://%s", domain)
	endpoint := fmt.Sprintf("/%s/json", url.PathEscape(topic))

	req, err := http.NewRequest(
		http.MethodGet,
		baseURL+endpoint,
		nil,
	)
	if err != nil {
		slog.Error("error creating ntfy request", "err", err)
		return nil, err
	}

	if n.auth != "" {
		req.Header.Add("Authorization", "Bearer "+n.auth)
	}

	resp, err := n.client.Do(req)
	if err != nil {
		slog.Error("error connecting to ntfy server", "err", err)
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		if closeErr := resp.Body.Close(); closeErr != nil {
			slog.Debug("failed to close response body", "err", closeErr)
		}
		slog.Error("invalid status code", "expected", http.StatusOK, "domain", n.domain, "statusCode", resp.StatusCode)
		return nil, fmt.Errorf("invalid response code from ntfy: %d", resp.StatusCode)
	}

	return resp.Body, nil
}
