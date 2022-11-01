# ntfy-to-slack

Rudimentary Go daemon to subscribe to a Ntfy topic and send the messages to a Slack webhook.

## Instructions (Linux/macOS/Windows docker)

1. ```git clone https://github.com/ozskywalker/ntfy-to-slack```
2. ```docker build -t ozskywalker/ntfy-to-slack```
3. ```
   docker run --env="NTFY_DOMAIN=<my-ntfy-server>" --env="NTFY_TOPIC=<my-ntfy-topic>" --env="SLACK_WEBHOOK_URL=<my-slack-webhook>" -d --restart always ozskywalker/ntfy-to-slack:latest
   ```

## Instructions (regular binary)

1. ```git clone https://github.com/ozskywalker/ntfy-to-slack```
2. ```go build .```

Run the resulting binary at your own leisure, with either environment variables or flags to specify configuration.