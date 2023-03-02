# ntfy-to-slack

Rudimentary Go daemon to subscribe to a Ntfy topic and send the messages to a Slack webhook.

## Instructions (Linux/macOS/Windows docker)

1. ```git clone https://github.com/ozskywalker/ntfy-to-slack```
2. ```cd ntfy-to-slack```
3. ```docker build -t ozskywalker/ntfy-to-slack .```
4. ```
   docker run --env="NTFY_DOMAIN=<my-ntfy-server>" --env="NTFY_TOPIC=<my-ntfy-topic>" --env="SLACK_WEBHOOK_URL=<my-slack-webhook>" --env="NTFY_AUTH=<token>" -d --restart always ozskywalker/ntfy-to-slack:latest
   ```

(NTFY_AUTH only required for topics requiring authentication.)

## Instructions (regular binary)

1. ```git clone https://github.com/ozskywalker/ntfy-to-slack```
2. ```cd ntfy-to-slack```
3. ```go build .```

Run the resulting binary at your own leisure, with either environment variables or flags to specify configuration.