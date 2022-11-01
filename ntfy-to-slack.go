package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	slack "github.com/ashwanthkumar/slack-go-webhook"
	"log"
	"net/http"
	"os"
	"time"
)

const VERSION = "v1 2022-10-31"
const UpstreamNtfyServer = "ntfy.sh"

var defaultNtfyDomain = UpstreamNtfyServer
var ntfyDomain *string
var ntfyTopic *string
var slackWebhookUrl *string

type NtfyMessage struct {
	Id      string
	Time    int64
	Event   string
	Topic   string
	Title   string
	Message string
}

func sendToSlack(message string) {
	payload := slack.Payload{
		Text: "(" + *ntfyTopic + ") " + message,
	}

	err := slack.Send(*slackWebhookUrl, "", payload)
	if len(err) > 0 {
		fmt.Printf("error: %s\n", err)
	}
}

func main() {
	var envNtfyDomain, ok = os.LookupEnv("NTFY_DOMAIN")
	if ok {
		defaultNtfyDomain = envNtfyDomain
	}
	envNtfyTopic, ok := os.LookupEnv("NTFY_TOPIC")
	envSlackWebhookUrl, ok := os.LookupEnv("SLACK_WEBHOOK_URL")

	ntfyDomain = flag.String("ntfy-domain", defaultNtfyDomain, "Choose the ntfy server to interact with.\nDefaults to "+UpstreamNtfyServer+" or the value of the NTFY_DOMAIN env var, if it is set")
	ntfyTopic = flag.String("ntfy-topic", envNtfyTopic, "Choose the ntfy topic to interact with\nDefaults to the value of the NTFY_TOPIC env var, if it is set")
	slackWebhookUrl = flag.String("slack-webhook", envSlackWebhookUrl, "Choose the slack webhook url to send messages to\nDefaults to the value of the SLACK_WEBHOOK_URL env var, if it is set")
	version := flag.Bool("v", false, "prints current ntfy-to-slack version")

	flag.Parse()

	if *version {
		println(VERSION)
		os.Exit(0)
	}

	resp, err := http.Get("https://" + *ntfyDomain + "/" + *ntfyTopic + "/json")
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	var msg NtfyMessage

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		err := json.Unmarshal([]byte(scanner.Text()), &msg)
		if err != nil {
			println(err)
			fmt.Printf("while processing %s", scanner.Text())
			sendToSlack("bot error: " + err.Error())
		}

		timeT := time.Unix(msg.Time, 0).String()

		switch msg.Event {
		case "open":
			fmt.Printf("%s: %s subscription established\n", timeT, *ntfyDomain)
			sendToSlack("bot restarted; " + *ntfyDomain + " subscription established")
		case "keepalive":
			fmt.Printf("%s: keepalive\n", timeT)
		case "message":
			{
				fmt.Printf("%s: sending to Slack: %s / %s\n", timeT, msg.Title, msg.Message)
				sendToSlack(msg.Title + ": " + msg.Message)
			}
		default:
			fmt.Printf("bad message received: %s\n", scanner.Text())
		}
	}
}
