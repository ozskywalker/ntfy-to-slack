package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"time"
)

// many thanks to github.com/schaluerlauer for showing me the errors of my early ways with golang
// even if this utility has overlived its usefulness, education is never wasted

const (
	version            = "v1.4 2025-04-19"
	upstreamNtfyServer = "ntfy.sh"
)

var (
	defaultNtfyDomain = upstreamNtfyServer
	ntfyDomain        *string
	ntfyTopic         *string
	ntfyAuth          *string
	slackWebhookUrl   *string
)

type ntfyMessage struct {
	Id      string
	Time    int64
	Event   string
	Topic   string
	Title   string
	Message string
}

type slackMessage struct {
	Text string `json:"text"`
}

func main() {
	// Setup logging based on environment
	if logLevel, ok := os.LookupEnv("LOG_LEVEL"); ok {
		switch logLevel {
		case "debug":
			slog.SetLogLoggerLevel(slog.LevelDebug)
		case "warn":
			slog.SetLogLoggerLevel(slog.LevelWarn)
		case "error":
			slog.SetLogLoggerLevel(slog.LevelError)
		default:
			slog.SetLogLoggerLevel(slog.LevelInfo)
		}
	} else {
		slog.SetLogLoggerLevel(slog.LevelInfo)
	}

	// Setup environment variables and flags
	var envNtfyDomain, ok = os.LookupEnv("NTFY_DOMAIN")
	if ok {
		defaultNtfyDomain = envNtfyDomain
	}
	envNtfyTopic, _ := os.LookupEnv("NTFY_TOPIC")
	envNtfyAuth, _ := os.LookupEnv("NTFY_AUTH")
	envSlackWebhookUrl, _ := os.LookupEnv("SLACK_WEBHOOK_URL")

	ntfyDomain = flag.String("ntfy-domain", defaultNtfyDomain, "Choose the ntfy server to interact with.\nDefaults to "+upstreamNtfyServer+" or the value of the NTFY_DOMAIN env var, if it is set")
	ntfyTopic = flag.String("ntfy-topic", envNtfyTopic, "Choose the ntfy topic to interact with\nDefaults to the value of the NTFY_TOPIC env var, if it is set")
	ntfyAuth = flag.String("ntfy-auth", envNtfyAuth, "Specify token for reserved topics")
	slackWebhookUrl = flag.String("slack-webhook", envSlackWebhookUrl, "Choose the slack webhook url to send messages to\nDefaults to the value of the SLACK_WEBHOOK_URL env var, if it is set")
	versionFlag := flag.Bool("v", false, "prints current ntfy-to-slack version")

	flag.Parse()

	// Print help if no arguments provided (and no required env vars set)
	if len(os.Args) == 1 && envNtfyTopic == "" && envSlackWebhookUrl == "" {
		fmt.Println("ntfy-to-slack", version)
		fmt.Println("Forwards ntfy.sh messages to Slack")
		fmt.Println()
		flag.Usage()
		os.Exit(1)
	}

	if *versionFlag {
		println(version)
		os.Exit(0)
	}

	// Validate required parameters
	if *ntfyTopic == "" {
		fmt.Fprintln(os.Stderr, "Error: ntfy topic is required")
		flag.Usage()
		os.Exit(1)
	}

	// Validate domain
	if _, err := validateDomain(*ntfyDomain); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Validate topic
	if _, err := validateTopic(*ntfyTopic); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Validate slack webhook URL
	if *slackWebhookUrl == "" {
		fmt.Fprintln(os.Stderr, "Error: Slack webhook URL is required")
		flag.Usage()
		os.Exit(1)
	}

	// Basic validation of webhook URL format
	webhookURL, err := url.Parse(*slackWebhookUrl)
	if err != nil || webhookURL.Scheme != "https" || webhookURL.Host == "" {
		fmt.Fprintln(os.Stderr, "Error: Invalid Slack webhook URL format. Must be a valid HTTPS URL")
		os.Exit(1)
	}

	for {
		if err := waitForNtfyMessage(); err != nil {
			slog.Error("waitForNtfyMessage", "err", err)
			os.Exit(1)
		} else {
			slog.Info("connection closed, restarting")
		}
		time.Sleep(30 * time.Second)
	}
}

func validateDomain(domain string) (string, error) {
	domainPattern := regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9-]{0,61}[a-zA-Z0-9](?:\.[a-zA-Z]{2,})+$`)
	if !domainPattern.MatchString(domain) {
		return "", fmt.Errorf("invalid domain format: %s", domain)
	}
	return domain, nil
}

func validateTopic(topic string) (string, error) {
	topicPattern := regexp.MustCompile(`^[a-zA-Z0-9_-]{1,64}$`)
	if !topicPattern.MatchString(topic) {
		return "", fmt.Errorf("invalid topic format: %s", topic)
	}
	return topic, nil
}

func waitForNtfyMessage() error {
	// Validate domain and topic
	domain, err := validateDomain(*ntfyDomain)
	if err != nil {
		return err
	}

	topic, err := validateTopic(*ntfyTopic)
	if err != nil {
		return err
	}

	baseURL := fmt.Sprintf("https://%s", domain)
	endpoint := fmt.Sprintf("/%s/json", url.PathEscape(topic))

	client := &http.Client{}
	req, err := http.NewRequest(
		http.MethodGet,
		baseURL+endpoint,
		nil,
	)
	if err != nil {
		slog.Error("error getting ntfy response", "err", err)
		return err
	}
	if ntfyAuth != nil {
		req.Header.Add("Authorization", "Bearer "+*ntfyAuth)
	}

	resp, err := client.Do(req)
	if err != nil {
		slog.Error("error connecting to ntfy server", "err", err)
		return err
	} else if resp.StatusCode != http.StatusOK {
		slog.Error("invalid status code", "expected", http.StatusOK, "domain", *ntfyDomain, "statusCode", strconv.FormatInt(int64(resp.StatusCode), 10))
		return errors.New("invalid response code from ntfy")
	}
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		var msg ntfyMessage
		err := json.Unmarshal([]byte(scanner.Text()), &msg)
		if err != nil {
			slog.Error("error while processing ntfy message", "err", err, "text", scanner.Text())
			continue
		}

		switch msg.Event {
		case "open":
			slog.Info("subscription established", "domain", *ntfyDomain)
			continue
		case "keepalive":
			slog.Debug("keepalive")
			continue
		case "message":
			slog.Info("sending message", "title", msg.Title, "message", msg.Message)
			if msg.Title != "" {
				if err := sendToSlack(&slackMessage{
					Text: "**" + msg.Title + "**: " + msg.Message,
				}); err != nil {
					slog.Error("error sending message", "err", err)
				}
			} else {
				if err := sendToSlack(&slackMessage{
					Text: msg.Message,
				}); err != nil {
					slog.Error("error sending message", "err", err)
				}
			}
			continue
		default:
			slog.Warn("bad message received", "message", scanner.Text())
			continue
		}
	}

	return nil
}

func sendToSlack(webhook *slackMessage) error {
	if webhook == nil {
		return errors.New("webhook undefined")
	}

	jsonBytes, err := json.Marshal(webhook)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(
		http.MethodPost,
		*slackWebhookUrl,
		bytes.NewBuffer(jsonBytes),
	)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Timeout: 3 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func(Body io.ReadCloser) {
		if err := Body.Close(); err != nil {
			slog.Error("error closing response body", "err", err)
		}
	}(resp.Body)

	if body, err := io.ReadAll(resp.Body); err != nil {
		slog.Error("error parsing body", "err", err)
		return err
	} else {
		slog.Debug("slack response", "status", resp.StatusCode, "body", body)
	}

	if resp.StatusCode >= 400 {
		return errors.New("error status code " + strconv.FormatInt(int64(resp.StatusCode), 10))
	}

	return nil
}
