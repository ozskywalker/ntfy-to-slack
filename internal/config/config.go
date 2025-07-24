package config

import (
	"flag"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strconv"
)

// Provider interface for application configuration
type Provider interface {
	GetNtfyDomain() string
	GetNtfyTopic() string
	GetNtfyAuth() string
	GetSlackWebhookURL() string
	GetPostProcessWebhook() string
	GetPostProcessTemplateFile() string
	GetPostProcessTemplate() string
	GetWebhookTimeoutSeconds() int
	GetWebhookRetries() int
	GetWebhookMaxResponseSizeMB() int
	Validate() error
}

// Config holds all application configuration
type Config struct {
	NtfyDomain               string
	NtfyTopic                string
	NtfyAuth                 string
	SlackWebhookURL          string
	LogLevel                 string
	ShowVersion              bool
	ShowHelp                 bool
	PostProcessWebhook       string
	PostProcessTemplateFile  string
	PostProcessTemplate      string
	WebhookTimeoutSeconds    int
	WebhookRetries           int
	WebhookMaxResponseSizeMB int
}

// New creates a new configuration from command line args and environment
func New(args []string) (*Config, error) {
	config := &Config{}
	
	// Create a new flag set to avoid global state
	fs := flag.NewFlagSet("ntfy-to-slack", flag.ContinueOnError)
	
	// Get environment variables
	envNtfyDomain := getEnvOrDefault("NTFY_DOMAIN", "ntfy.sh")
	envNtfyTopic := os.Getenv("NTFY_TOPIC")
	envNtfyAuth := os.Getenv("NTFY_AUTH")
	envSlackWebhookURL := os.Getenv("SLACK_WEBHOOK_URL")
	envLogLevel := getEnvOrDefault("LOG_LEVEL", "info")
	envPostProcessWebhook := os.Getenv("POST_PROCESS_WEBHOOK")
	envPostProcessTemplateFile := os.Getenv("POST_PROCESS_TEMPLATE_FILE")
	envPostProcessTemplate := os.Getenv("POST_PROCESS_TEMPLATE")
	envWebhookTimeoutSeconds := getEnvOrDefault("WEBHOOK_TIMEOUT_SECONDS", "30")
	envWebhookRetries := getEnvOrDefault("WEBHOOK_RETRIES", "3")
	envWebhookMaxResponseSizeMB := getEnvOrDefault("WEBHOOK_MAX_RESPONSE_SIZE_MB", "1")
	
	// Define flags
	fs.StringVar(&config.NtfyDomain, "ntfy-domain", envNtfyDomain, "Choose the ntfy server to interact with")
	fs.StringVar(&config.NtfyTopic, "ntfy-topic", envNtfyTopic, "Choose the ntfy topic to interact with")
	fs.StringVar(&config.NtfyAuth, "ntfy-auth", envNtfyAuth, "Specify token for reserved topics")
	fs.StringVar(&config.SlackWebhookURL, "slack-webhook", envSlackWebhookURL, "Choose the slack webhook url to send messages to")
	fs.StringVar(&config.PostProcessWebhook, "post-process-webhook", envPostProcessWebhook, "Webhook URL for post-processing messages")
	fs.StringVar(&config.PostProcessTemplateFile, "post-process-template-file", envPostProcessTemplateFile, "Template file path for post-processing messages")
	fs.StringVar(&config.PostProcessTemplate, "post-process-template", envPostProcessTemplate, "Inline template for post-processing messages")
	fs.IntVar(&config.WebhookTimeoutSeconds, "webhook-timeout", 0, "Webhook timeout in seconds (default: 30)")
	fs.IntVar(&config.WebhookRetries, "webhook-retries", 0, "Number of webhook retries (default: 3)")
	fs.IntVar(&config.WebhookMaxResponseSizeMB, "webhook-max-response-size", 0, "Maximum webhook response size in MB (default: 1)")
	fs.BoolVar(&config.ShowVersion, "v", false, "prints current ntfy-to-slack version")
	config.LogLevel = envLogLevel
	
	// Set defaults for webhook configuration
	if timeoutSeconds, err := strconv.Atoi(envWebhookTimeoutSeconds); err == nil {
		config.WebhookTimeoutSeconds = timeoutSeconds
	} else {
		config.WebhookTimeoutSeconds = 30
	}
	
	if retries, err := strconv.Atoi(envWebhookRetries); err == nil {
		config.WebhookRetries = retries
	} else {
		config.WebhookRetries = 3
	}
	
	if maxSize, err := strconv.Atoi(envWebhookMaxResponseSizeMB); err == nil {
		config.WebhookMaxResponseSizeMB = maxSize
	} else {
		config.WebhookMaxResponseSizeMB = 1
	}
	
	// Parse arguments
	err := fs.Parse(args)
	if err != nil {
		return nil, fmt.Errorf("failed to parse flags: %w", err)
	}
	
	// Check for help request (no args and no required env vars)
	if len(args) == 0 && envNtfyTopic == "" && envSlackWebhookURL == "" {
		config.ShowHelp = true
	}
	
	return config, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.ShowVersion || c.ShowHelp {
		return nil // Skip validation for help/version
	}
	
	// Validate required parameters
	if c.NtfyTopic == "" {
		return fmt.Errorf("ntfy topic is required")
	}
	
	if c.SlackWebhookURL == "" {
		return fmt.Errorf("Slack webhook URL is required")
	}
	
	// Validate domain
	if _, err := ValidateDomain(c.NtfyDomain); err != nil {
		return fmt.Errorf("invalid domain: %w", err)
	}
	
	// Validate topic
	if _, err := ValidateTopic(c.NtfyTopic); err != nil {
		return fmt.Errorf("invalid topic: %w", err)
	}
	
	// Validate slack webhook URL
	webhookURL, err := url.Parse(c.SlackWebhookURL)
	if err != nil || webhookURL.Scheme != "https" || webhookURL.Host == "" {
		return fmt.Errorf("invalid Slack webhook URL format. Must be a valid HTTPS URL")
	}
	
	// Validate post-processing options (only one allowed)
	postProcessCount := 0
	if c.PostProcessWebhook != "" {
		postProcessCount++
	}
	if c.PostProcessTemplateFile != "" {
		postProcessCount++
	}
	if c.PostProcessTemplate != "" {
		postProcessCount++
	}
	
	if postProcessCount > 1 {
		return fmt.Errorf("only one post-processing option can be specified: webhook, template file, or inline template")
	}
	
	// Validate webhook URL if specified
	if c.PostProcessWebhook != "" {
		webhookURL, err := url.Parse(c.PostProcessWebhook)
		if err != nil || (webhookURL.Scheme != "http" && webhookURL.Scheme != "https") || webhookURL.Host == "" {
			return fmt.Errorf("invalid post-process webhook URL format. Must be a valid HTTP/HTTPS URL")
		}
	}
	
	// Validate webhook configuration values only if webhook is configured
	if c.PostProcessWebhook != "" {
		if c.WebhookTimeoutSeconds < 1 || c.WebhookTimeoutSeconds > 300 {
			return fmt.Errorf("webhook timeout must be between 1 and 300 seconds, got %d", c.WebhookTimeoutSeconds)
		}
		
		if c.WebhookRetries < 0 || c.WebhookRetries > 10 {
			return fmt.Errorf("webhook retries must be between 0 and 10, got %d", c.WebhookRetries)
		}
		
		if c.WebhookMaxResponseSizeMB < 1 || c.WebhookMaxResponseSizeMB > 100 {
			return fmt.Errorf("webhook max response size must be between 1 and 100 MB, got %d", c.WebhookMaxResponseSizeMB)
		}
	}
	
	return nil
}

// GetNtfyDomain implements ConfigProvider interface
func (c *Config) GetNtfyDomain() string {
	return c.NtfyDomain
}

// GetNtfyTopic implements ConfigProvider interface
func (c *Config) GetNtfyTopic() string {
	return c.NtfyTopic
}

// GetNtfyAuth implements ConfigProvider interface
func (c *Config) GetNtfyAuth() string {
	return c.NtfyAuth
}

// GetSlackWebhookURL implements ConfigProvider interface
func (c *Config) GetSlackWebhookURL() string {
	return c.SlackWebhookURL
}

// GetPostProcessWebhook implements ConfigProvider interface
func (c *Config) GetPostProcessWebhook() string {
	return c.PostProcessWebhook
}

// GetPostProcessTemplateFile implements ConfigProvider interface
func (c *Config) GetPostProcessTemplateFile() string {
	return c.PostProcessTemplateFile
}

// GetPostProcessTemplate implements ConfigProvider interface
func (c *Config) GetPostProcessTemplate() string {
	return c.PostProcessTemplate
}

// GetWebhookTimeoutSeconds implements ConfigProvider interface
func (c *Config) GetWebhookTimeoutSeconds() int {
	return c.WebhookTimeoutSeconds
}

// GetWebhookRetries implements ConfigProvider interface
func (c *Config) GetWebhookRetries() int {
	return c.WebhookRetries
}

// GetWebhookMaxResponseSizeMB implements ConfigProvider interface
func (c *Config) GetWebhookMaxResponseSizeMB() int {
	return c.WebhookMaxResponseSizeMB
}

// getEnvOrDefault returns environment variable value or default if not set
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func ValidateDomain(domain string) (string, error) {
	domainPattern := regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\.[a-zA-Z]{2,})+$`)
	if !domainPattern.MatchString(domain) {
		return "", fmt.Errorf("invalid domain format: %s", domain)
	}
	return domain, nil
}

func ValidateTopic(topic string) (string, error) {
	topicPattern := regexp.MustCompile(`^[a-zA-Z0-9_-]{1,64}$`)
	if !topicPattern.MatchString(topic) {
		return "", fmt.Errorf("invalid topic format: %s", topic)
	}
	return topic, nil
}