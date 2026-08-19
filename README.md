# ntfy-to-slack

ntfy-to-slack forwards your [ntfy.sh](https://ntfy.sh) notifications into your favorite Slack channel.

Meant to run from a container you can set & forget.

[![Go Version](https://img.shields.io/badge/Go-1.26.6-blue.svg)](https://golang.org/doc/devel/release.html)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
![Claude Used](https://img.shields.io/badge/Claude-Used-4B5AEA)

## CI/CD Status

[![CI](https://github.com/ozskywalker/ntfy-to-slack/actions/workflows/test.yml/badge.svg?branch=main)](https://github.com/ozskywalker/ntfy-to-slack/actions/workflows/test.yml)
[![Release](https://github.com/ozskywalker/ntfy-to-slack/actions/workflows/release.yml/badge.svg)](https://github.com/ozskywalker/ntfy-to-slack/actions/workflows/release.yml)
[![Coverage](https://codecov.io/gh/ozskywalker/ntfy-to-slack/branch/main/graph/badge.svg)](https://codecov.io/gh/ozskywalker/ntfy-to-slack)
[![Go Report Card](https://goreportcard.com/badge/github.com/ozskywalker/ntfy-to-slack)](https://goreportcard.com/report/github.com/ozskywalker/ntfy-to-slack)

## Features

**Post-processing support:** Transform messages with a Go [`text/template`](https://pkg.go.dev/text/template) template, or call an external service via Webhook (like N8N), before passing the transformed result to Slack.

 - **Resilient by design:** Automatically reconnects on any connection failure (network errors, non-200 responses, or a closed stream) with a 30-second retry interval, resuming from the last message seen so a reconnect doesn't lose messages sent during the gap, and keeps processing subsequent messages despite individual message errors. A transient Slack failure (429 or 5xx, honoring `Retry-After`) is retried instead of dropping the message.
 - **Shuts down cleanly:** Responds to SIGTERM/SIGINT (e.g. `docker stop`) by stopping immediately instead of finishing out a reconnect wait or being force-killed after the grace period
 - **Detects a silently dead connection:** If no data (including ntfy's own ~45-second keepalives) arrives for 2 minutes, forces a reconnect instead of sitting on a stalled connection indefinitely with no indication anything is wrong
 - **Structured logging with configurable levels:** Contextual logging with relevant metadata (domains, topics, error details) and configurable log levels (debug/info/warn/error) for better debugging and monitoring
 - **Modular, interface-driven design** for testability and maintainability

## Installation

### Pre-built Binaries (Recommended)

Download the latest release from the [GitHub Releases page](https://github.com/ozskywalker/ntfy-to-slack/releases):

```bash
# Linux/macOS example
curl -L https://github.com/ozskywalker/ntfy-to-slack/releases/latest/download/ntfy-to-slack-Linux-x86_64.tar.gz | tar xz
chmod +x ntfy-to-slack

# Windows (PowerShell)
Invoke-WebRequest -Uri "https://github.com/ozskywalker/ntfy-to-slack/releases/latest/download/ntfy-to-slack-Windows-x86_64.zip" -UseBasicParsing -OutFile "ntfy-to-slack.zip"
Expand-Archive -Path "ntfy-to-slack.zip" -DestinationPath "."

# Verify installation
./ntfy-to-slack -v
```

### Using Docker

```bash
# Clone this repo & build the container image
git clone https://github.com/ozskywalker/ntfy-to-slack && cd ntfy-to-slack && docker build -t ozskywalker/ntfy-to-slack .

# Start ntfy-to-slack
docker run --env="NTFY_DOMAIN=ntfy.sh" \
           --env="NTFY_TOPIC=your-topic" \
           --env="SLACK_WEBHOOK_URL=https://hooks.slack.com/your-webhook" \
           --env="NTFY_AUTH=your-token" \  # Optional
           --env="LOG_LEVEL=info" \        # Optional
           -d --restart always \
           ozskywalker/ntfy-to-slack:latest
```

### Health Checks

Setting `--health-addr`/`HEALTH_ADDR` (e.g. `:8080`) serves a `/healthz`
liveness endpoint: `200 {"status":"ok"}` while the app is making forward
progress, `503 {"status":"unhealthy"}` if it's gone silent for an unusually
long time. This reflects the app's own liveness, not ntfy's reachability --
an ntfy outage the app is already correctly retrying through still reports
healthy, since restarting the container wouldn't fix an outage it doesn't
control. Disabled unless set, so existing deployments don't suddenly bind a
port they didn't ask for.

```bash
docker run --env="NTFY_TOPIC=your-topic" \
           --env="SLACK_WEBHOOK_URL=https://hooks.slack.com/your-webhook" \
           --env="HEALTH_ADDR=:8080" \
           -p 8080:8080 \
           --health-cmd="wget -qO- http://localhost:8080/healthz || exit 1" \
           -d --restart always \
           ozskywalker/ntfy-to-slack:latest
```

### Build from Source

Requires Go 1.26+ to be pre-installed.

```bash
# Clone repository
git clone https://github.com/ozskywalker/ntfy-to-slack
cd ntfy-to-slack

# Build the binary
go build -v ./cmd/ntfy-to-slack

# Run with command line flags
./ntfy-to-slack --ntfy-topic=your-topic --slack-webhook=https://hooks.slack.com/your-webhook
```

### Post-Processing Examples

Templates use standard Go [`text/template`](https://pkg.go.dev/text/template) syntax (`{{.Title}}`, `{{.Message}}`, `{{if}}`, `{{range}}`, ...), not Mustache — despite the similar `{{ }}` delimiters, the two template languages are not compatible.

Templates and webhook payloads receive the full ntfy message, matching [ntfy's own JSON format](https://docs.ntfy.sh/subscribe/api/#json-message-format):

| Field        | Type            | Notes                                                       |
|--------------|-----------------|--------------------------------------------------------------|
| `Id`         | string          |                                                                |
| `Time`       | int64           | Unix timestamp                                                |
| `Expires`    | int64           | Unix timestamp; 0 if the message doesn't expire               |
| `Event`      | string          | `open`, `keepalive`, or `message`                              |
| `Topic`      | string          |                                                                |
| `Title`      | string          |                                                                |
| `Message`    | string          |                                                                |
| `Priority`   | int             | 1 (min) to 5 (max); 0 if unset (ntfy's default priority is 3) |
| `Tags`       | []string        |                                                                |
| `Click`      | string          | URL opened when the notification is clicked                   |
| `Actions`    | []NtfyAction    | `{{range .Actions}}{{.Label}}: {{.URL}}{{end}}`                |
| `Attachment` | *NtfyAttachment | `{{if .Attachment}}{{.Attachment.Name}}: {{.Attachment.URL}}{{end}}` |

**In-line template formatting:**
```bash
./ntfy-to-slack --ntfy-topic alerts --slack-webhook https://hooks.slack.com/... --post-process-template "🚨 *{{.Title}}* Alert\n📄{{.Message}}\n⏰ Time: {{.Time}}"
```

**Webhook integration with N8N:**
```bash
./ntfy-to-slack --ntfy-topic monitoring --slack-webhook https://hooks.slack.com/... --post-process-webhook https://n8n.yourcompany.com/webhook/ntfy-processor
```

**Template file for complex formatting:**
```bash
./ntfy-to-slack --ntfy-topic alerts --slack-webhook https://hooks.slack.com/... --post-process-template-file /path/to/alert-template.tmpl
```

## Configuration

ntfy-to-slack can be configured using either environment variables or command-line flags:

| Environment Variable | Flag             | Description                       | Default  | Required |
|----------------------|------------------|-----------------------------------|----------|----------|
| `NTFY_DOMAIN`        | `--ntfy-domain`  | ntfy server to connect to         | ntfy.sh  | No       |
| `NTFY_TOPIC`         | `--ntfy-topic`   | ntfy topic to subscribe to        | -        | Yes      |
| `NTFY_AUTH`          | `--ntfy-auth`    | Bearer token for reserved topics (mutually exclusive with `NTFY_USERNAME`/`NTFY_PASSWORD`) | - | No       |
| `NTFY_USERNAME`      | `--ntfy-username`| Username for HTTP Basic auth on reserved topics (requires `NTFY_PASSWORD`) | - | No |
| `NTFY_PASSWORD`      | `--ntfy-password`| Password for HTTP Basic auth on reserved topics (requires `NTFY_USERNAME`) | - | No |
| `SLACK_WEBHOOK_URL`  | `--slack-webhook`| Slack webhook URL                 | -        | Yes      |
| `POST_PROCESS_WEBHOOK` | `--post-process-webhook` | Webhook URL for post-processing | - | No |
| `POST_PROCESS_TEMPLATE_FILE` | `--post-process-template-file` | Template file path for post-processing | - | No |
| `POST_PROCESS_TEMPLATE` | `--post-process-template` | Inline template for post-processing | - | No |
| `WEBHOOK_TIMEOUT_SECONDS` | `--webhook-timeout` | Webhook timeout in seconds (1-300) | 30 | No |
| `WEBHOOK_RETRIES` | `--webhook-retries` | Number of webhook retries (0-10) | 3 | No |
| `WEBHOOK_MAX_RESPONSE_SIZE_MB` | `--webhook-max-response-size` | Max webhook response size in MB (1-100) | 1 | No |
| `LOG_LEVEL`          | -                | Log level (debug/info/warn/error) | info     | No       |
| `HEALTH_ADDR`        | `--health-addr`  | Address to serve a `/healthz` liveness endpoint on, e.g. `:8080` | - (disabled) | No |

Command-line flags take precedence over environment variables.

**Note**: Only one post-processing option can be specified at a time. Webhook configuration options (`WEBHOOK_TIMEOUT_SECONDS`, `WEBHOOK_RETRIES`, `WEBHOOK_MAX_RESPONSE_SIZE_MB`) only apply to webhook post-processing, and not the Slack webhook.

### Configuration Examples

**Basic usage:**
```bash
./ntfy-to-slack --ntfy-topic alerts --slack-webhook https://hooks.slack.com/...
```

**With in-line template formatting:**
```bash
./ntfy-to-slack --ntfy-topic alerts --slack-webhook https://hooks.slack.com/... \
  --post-process-template "🚨 *{{.Title}}* Alert\n📄 {{.Message}}\n⏰ Time: {{.Time}}"
```

**With template file for complex formatting:**
```bash
./ntfy-to-slack --ntfy-topic alerts --slack-webhook https://hooks.slack.com/... \
  --post-process-template-file /path/to/alert-template.tmpl
```

**With webhook integration:**
```bash
./ntfy-to-slack --ntfy-topic monitoring --slack-webhook https://hooks.slack.com/... \
  --post-process-webhook https://n8n.yourcompany.com/webhook/ntfy-processor
```

# Development

## Architecture

```
├── cmd/
│   └── ntfy-to-slack/
│       ├── main.go               # Main entry point
│       └── main_test.go
├── internal/                     # Internal packages (not importable externally)
│   ├── app/
│   │   ├── app.go                # Application orchestration
│   │   ├── idlewatchdog.go       # Dead-connection detection
│   │   └── *_test.go
│   ├── config/
│   │   ├── config.go             # Configuration management
│   │   ├── postprocessor.go      # Post-processing (templates & webhooks)
│   │   └── *_test.go
│   ├── ntfy/
│   │   ├── ntfy.go               # Ntfy client
│   │   └── *_test.go
│   ├── processor/
│   │   ├── processor.go          # Message processing
│   │   ├── interfaces.go         # Clean interface definitions
│   │   └── *_test.go
│   ├── slack/
│   │   ├── slack.go              # Slack integration
│   │   └── *_test.go
│   ├── testutil/
│   │   └── mocks.go              # Test doubles shared across packages' tests
│   └── version/
│       ├── version.go
│       └── version_test.go
├── Makefile                      # Test automation
└── .github/workflows/            # CI/CD pipeline
```

Tests live beside the code they cover (Go's standard layout), rather than in a
separate `tests/` tree -- this lets a test reach unexported internals when it
needs to, and `go test ./...` reports accurate per-package coverage with no
extra flags.

## Testing

This project includes a comprehensive test suite covering unit tests, integration-style tests, and HTTP interactions.

### Running Tests

```bash
# Run all tests
go test -v ./...

# Run tests with coverage
go test -v -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html

# Run just one package's tests
go test -v ./internal/config/...
```

### Using Make (if available)

```bash
# Run all tests
make test

# Run tests with coverage report
make test-coverage

# Run full build pipeline
make all
```

## Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details on:

- Using conventional commits for automated changelog generation
- Development workflow and code guidelines  
- Pull request process

For major changes, please open an issue first to discuss what you would like to change.

## Troubleshooting

- If you see "invalid domain format" or "invalid topic format" errors, check that your domain and topic follow the expected format patterns
- If connection to ntfy fails, verify your internet connection and that the ntfy server is accessible
- For authentication issues with reserved topics, ensure your `NTFY_AUTH` token is correct
- Check Slack webhook URL format if you receive webhook errors

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- Thanks to [ntfy.sh](https://ntfy.sh) for the excellent notification service
- Special thanks to [@schlauerlauer](https://github.com/schlauerlauer) for some guidance thru his fork
