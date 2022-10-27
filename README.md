# ntfy-to-slack

Rudimentary Go daemon to subscribe to a Ntfy topic and send the messages to a Slack webhook.

## Instructions (Linux/macOS/Windows docker)

1. ```git clone https://github.com/ozskywalker/ntfy-to-slack```
2. Edit ntfy-to-slack.go, replacing Topic & Webhook URL constants with appropriate URLs
3. ```docker build -t ozskywalker/ntfy-to-slack```
4. ```docker run -d --restart always ozskywalker/ntfy-to-slack:latest```

## Instructions (regular binary)

1. ```git clone https://github.com/ozskywalker/ntfy-to-slack```
2. Edit ntfy-to-slack.go, replacing Topic & Webhook URL constants with appropriate URLs
3. ```go build .```

Run the resulting binary at your own lesiure.