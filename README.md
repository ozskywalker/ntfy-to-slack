# ntfy-to-slack

ntfy-to-slack forwards your [ntfy.sh](https://ntfy.sh) notifications into your favorite Slack channel.

Meant to run from a container you can set & forget.

[![Go Version](https://img.shields.io/badge/Go-1.24.4-blue.svg)](https://golang.org/doc/devel/release.html)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
![Claude Used](https://img.shields.io/badge/Claude-Used-4B5AEA)

## CI/CD Status

[![CI](https://github.com/ozskywalker/ntfy-to-slack/actions/workflows/test.yml/badge.svg?branch=main)](https://github.com/ozskywalker/ntfy-to-slack/actions/workflows/test.yml)
[![Release](https://github.com/ozskywalker/ntfy-to-slack/actions/workflows/release.yml/badge.svg)](https://github.com/ozskywalker/ntfy-to-slack/actions/workflows/release.yml)
[![Coverage](https://codecov.io/gh/ozskywalker/ntfy-to-slack/branch/main/graph/badge.svg)](https://codecov.io/gh/ozskywalker/ntfy-to-slack)
[![Go Report Card](https://goreportcard.com/badge/github.com/ozskywalker/ntfy-to-slack)](https://goreportcard.com/report/github.com/ozskywalker/ntfy-to-slack)

## Version 2 is here!

**Introducing Post-processing support:** Transform messages with a Mustache template, or call an external service via Webhook (like N8N), before passing the transformed result to Slack.

QoL changes:
 - **Improved error handling and resilience:** Implemented robust error recovery, automatic reconnection with 30-second intervals, graceful handling of network failures, and continued processing despite individual message errors
 - **Enhanced structured logging with configurable levels:** Added contextual logging with relevant metadata (domains, topics, error details) and configurable log levels (debug/info/warn/error) for better debugging and monitoring
 - **Refactoring:** Moved away from monolithic code into clean, modular components with a interface-driven design for improved testability and maintainability

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
./ntfy-to-slack --version
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

### Build from Source

Requires Go 1.24+ to be pre-installed.

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

For more details on Mustache templates, check out [the Mustache playground & documentation.](https://jgonggrijp.gitlab.io/wontache/playground.html)

**In-line template formatting:**
```bash
./ntfy-to-slack --ntfy-topic alerts --slack-webhook https://hooks.slack.com/... --post-process-template "🚨 **{{.Title}}** Alert\n📄{{.Message}}\n⏰ Time: {{.Time}}"
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
| `NTFY_AUTH`          | `--ntfy-auth`    | Authentication token for reserved topics | - | No       |
| `SLACK_WEBHOOK_URL`  | `--slack-webhook`| Slack webhook URL                 | -        | Yes      |
| `POST_PROCESS_WEBHOOK` | `--post-process-webhook` | Webhook URL for post-processing | - | No |
| `POST_PROCESS_TEMPLATE_FILE` | `--post-process-template-file` | Template file path for post-processing | - | No |
| `POST_PROCESS_TEMPLATE` | `--post-process-template` | Inline template for post-processing | - | No |
| `WEBHOOK_TIMEOUT_SECONDS` | `--webhook-timeout` | Webhook timeout in seconds (1-300) | 30 | No |
| `WEBHOOK_RETRIES` | `--webhook-retries` | Number of webhook retries (0-10) | 3 | No |
| `WEBHOOK_MAX_RESPONSE_SIZE_MB` | `--webhook-max-response-size` | Max webhook response size in MB (1-100) | 1 | No |
| `LOG_LEVEL`          | -                | Log level (debug/info/warn/error) | info     | No       |

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
  --post-process-template "🚨 **{{.Title}}** Alert\n📄 {{.Message}}\n⏰ Time: {{.Time}}"
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
│       └── main.go               # Main entry point
├── internal/                     # Internal packages (not importable externally)
│   ├── app/
│   │   └── app.go                # Application orchestration
│   ├── config/
│   │   ├── config.go             # Configuration management
│   │   └── postprocessor.go      # Post-processing (templates & webhooks)
│   ├── ntfy/
│   │   └── ntfy.go               # Ntfy client
│   ├── processor/
│   │   ├── processor.go          # Message processing
│   │   └── interfaces.go         # Clean interface definitions
│   └── slack/
│       └── slack.go              # Slack integration
├── tests/
│   ├── unit/                     # Unit tests
│   │   └── *_test.go             # Component-specific unit tests
│   └── integration/              # Integration tests
│       └── *_test.go             # End-to-end integration tests
├── Makefile                      # Test automation
└── .github/workflows/            # CI/CD pipeline
```

## Testing

This project includes a comprehensive test suite covering unit tests, integration tests, and HTTP interactions.

### Running Tests

```bash
# Run all tests
go test -v ./tests/...

# Run tests with coverage
go test -v -coverprofile=coverage.out -coverpkg=./... ./tests/...
go tool cover -html=coverage.out -o coverage.html

# Run only unit tests
go test -v ./tests/unit/...

# Run only integration tests
go test -v ./tests/integration/...
```

### Using Make (if available)

```bash
# Run all tests
make test

# Run tests with coverage report
make test-coverage

# Run only unit tests
make test-unit

# Run only integration tests
make test-integration

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
