# ntfy-to-slack

ntfy-to-slack forwards your [ntfy.sh](https://ntfy.sh) notifications into your favorite Slack channel.

Meant to run from a container you can set & forget.

[![Go Version](https://img.shields.io/badge/Go-1.24.4-blue.svg)](https://golang.org/doc/devel/release.html)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## Installation

### Using Docker (Recommended)

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

### Using CLI

Will require golang to be pre-installed.

```bash
# Clone repository
git clone https://github.com/ozskywalker/ntfy-to-slack
cd ntfy-to-slack

# Build the binary
go build .

# Run with command line flags
./ntfy-to-slack --ntfy-topic=your-topic --slack-webhook=https://hooks.slack.com/your-webhook
```

## Configuration

ntfy-to-slack can be configured using either environment variables or command-line flags:

| Environment Variable | Flag             | Description                       | Default  | Required |
|----------------------|------------------|-----------------------------------|----------|----------|
| `NTFY_DOMAIN`        | `--ntfy-domain`  | ntfy server to connect to         | ntfy.sh  | No       |
| `NTFY_TOPIC`         | `--ntfy-topic`   | ntfy topic to subscribe to        | -        | Yes      |
| `NTFY_AUTH`          | `--ntfy-auth`    | Authentication token for reserved topics | - | No       |
| `SLACK_WEBHOOK_URL`  | `--slack-webhook`| Slack webhook URL                 | -        | Yes      |
| `LOG_LEVEL`          | -                | Log level (debug/info/warn/error) | info     | No       |

Command-line flags take precedence over environment variables.

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
