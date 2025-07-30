package app

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/ozskywalker/ntfy-to-slack/internal/config"
	"github.com/ozskywalker/ntfy-to-slack/internal/ntfy"
	"github.com/ozskywalker/ntfy-to-slack/internal/processor"
	"github.com/ozskywalker/ntfy-to-slack/internal/slack"
)

// App represents the main application
type App struct {
	config     config.Provider
	ntfyClient ntfy.Client
	processor  processor.StreamProcessor
	version    string
}

// New creates a new application instance
func New(cfg config.Provider, version string) *App {
	// Create HTTP client
	httpClient := &http.Client{}

	// Create Slack sender
	slackSender := slack.NewSender(cfg.GetSlackWebhookURL(), httpClient)

	// Create post-processor if configured
	var postProcessor config.PostProcessor
	var err error

	if cfg.GetPostProcessWebhook() != "" {
		postProcessor = config.NewWebhookPostProcessorWithConfig(
			cfg.GetPostProcessWebhook(),
			cfg.GetWebhookTimeoutSeconds(),
			cfg.GetWebhookRetries(),
			cfg.GetWebhookMaxResponseSizeMB(),
		)
		slog.Info("webhook post-processor configured",
			"url", cfg.GetPostProcessWebhook(),
			"timeout", cfg.GetWebhookTimeoutSeconds(),
			"retries", cfg.GetWebhookRetries(),
			"max_response_size_mb", cfg.GetWebhookMaxResponseSizeMB())
	} else if cfg.GetPostProcessTemplateFile() != "" {
		postProcessor, err = config.NewMustachePostProcessorFromFile(cfg.GetPostProcessTemplateFile())
		if err != nil {
			slog.Error("failed to load template file, using default formatting", "file", cfg.GetPostProcessTemplateFile(), "err", err)
		} else {
			slog.Info("template file post-processor configured", "file", cfg.GetPostProcessTemplateFile())
		}
	} else if cfg.GetPostProcessTemplate() != "" {
		postProcessor, err = config.NewMustachePostProcessor(cfg.GetPostProcessTemplate())
		if err != nil {
			slog.Error("failed to parse inline template, using default formatting", "err", err)
		} else {
			slog.Info("inline template post-processor configured")
		}
	}

	// Create message processor
	var msgProcessor processor.StreamProcessor
	if postProcessor != nil {
		msgProcessor = processor.NewWithPostProcessor(slackSender, postProcessor)
	} else {
		msgProcessor = processor.New(slackSender)
	}

	// Create ntfy client
	ntfyClient := ntfy.NewClient(
		cfg.GetNtfyDomain(),
		cfg.GetNtfyTopic(),
		cfg.GetNtfyAuth(),
		httpClient,
	)

	return &App{
		config:     cfg,
		ntfyClient: ntfyClient,
		processor:  msgProcessor,
		version:    version,
	}
}

// Run starts the application main loop
func (a *App) Run() error {
	for {
		if err := a.runOnce(); err != nil {
			slog.Error("connection failed", "err", err)
			return err
		}

		slog.Info("connection closed, restarting in 30 seconds")
		time.Sleep(30 * time.Second)
	}
}

// runOnce performs a single connection attempt and message processing loop
func (a *App) runOnce() error {
	reader, err := a.ntfyClient.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to ntfy: %w", err)
	}
	defer reader.Close()

	slog.Info("connected to ntfy", "domain", a.config.GetNtfyDomain(), "topic", a.config.GetNtfyTopic())

	return a.processor.ProcessStream(reader)
}

// PrintHelp prints the application help message
func (a *App) PrintHelp() {
	fmt.Println("ntfy-to-slack", a.version)
	fmt.Println("Forwards ntfy.sh messages to Slack")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  ntfy-to-slack [flags]")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  --ntfy-domain string              Choose the ntfy server to interact with (default \"ntfy.sh\")")
	fmt.Println("  --ntfy-topic string               Choose the ntfy topic to interact with")
	fmt.Println("  --ntfy-auth string                Specify token for reserved topics")
	fmt.Println("  --slack-webhook string            Choose the slack webhook url to send messages to")
	fmt.Println("  --post-process-webhook string     Webhook URL for post-processing messages")
	fmt.Println("  --post-process-template-file path Template file for post-processing messages")
	fmt.Println("  --post-process-template string    Inline template for post-processing messages")
	fmt.Println("  --webhook-timeout int             Webhook timeout in seconds (default: 30)")
	fmt.Println("  --webhook-retries int             Number of webhook retries (default: 3)")
	fmt.Println("  --webhook-max-response-size int   Maximum webhook response size in MB (default: 1)")
	fmt.Println("  -v                                prints current ntfy-to-slack version")
	fmt.Println()
	fmt.Println("Post-Processing:")
	fmt.Println("  Only one post-processing option can be specified at a time.")
	fmt.Println("  Templates use Go template syntax with the ntfyMessage struct as context.")
	fmt.Println("  Webhooks receive POST requests with JSON payload and should return JSON or plain text.")
	fmt.Println()
	fmt.Println("Environment Variables:")
	fmt.Println("  NTFY_DOMAIN                ntfy server to connect to (default \"ntfy.sh\")")
	fmt.Println("  NTFY_TOPIC                 ntfy topic to subscribe to")
	fmt.Println("  NTFY_AUTH                  Authentication token for reserved topics")
	fmt.Println("  SLACK_WEBHOOK_URL          Slack webhook URL")
	fmt.Println("  POST_PROCESS_WEBHOOK       Webhook URL for post-processing")
	fmt.Println("  POST_PROCESS_TEMPLATE_FILE Template file path for post-processing")
	fmt.Println("  POST_PROCESS_TEMPLATE      Inline template for post-processing")
	fmt.Println("  WEBHOOK_TIMEOUT_SECONDS    Webhook timeout in seconds (default: 30)")
	fmt.Println("  WEBHOOK_RETRIES            Number of webhook retries (default: 3)")
	fmt.Println("  WEBHOOK_MAX_RESPONSE_SIZE_MB Maximum webhook response size in MB (default: 1)")
	fmt.Println("  LOG_LEVEL                  Log level (debug/info/warn/error) (default \"info\")")
}
