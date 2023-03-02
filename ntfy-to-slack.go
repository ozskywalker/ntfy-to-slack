package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	slack "github.com/ashwanthkumar/slack-go-webhook"
)

const VERSION = "v1.2 2023-03-01"
const UpstreamNtfyServer = "ntfy.sh"

var defaultNtfyDomain = UpstreamNtfyServer
var ntfyDomain *string
var ntfyTopic *string
var ntfyAuth *string
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

	if err := slack.Send(*slackWebhookUrl, "", payload); len(err) > 0 {
		log.Panic("sendToSlack: something went wrong", err[0])
	}
}

func main() {
	var envNtfyDomain, ok = os.LookupEnv("NTFY_DOMAIN")
	if ok {
		defaultNtfyDomain = envNtfyDomain
	}
	envNtfyTopic, ok := os.LookupEnv("NTFY_TOPIC")
	envNtfyAuth, ok := os.LookupEnv("NTFY_AUTH")
	envSlackWebhookUrl, ok := os.LookupEnv("SLACK_WEBHOOK_URL")

	ntfyDomain = flag.String("ntfy-domain", defaultNtfyDomain, "Choose the ntfy server to interact with.\nDefaults to "+UpstreamNtfyServer+" or the value of the NTFY_DOMAIN env var, if it is set")
	ntfyTopic = flag.String("ntfy-topic", envNtfyTopic, "Choose the ntfy topic to interact with\nDefaults to the value of the NTFY_TOPIC env var, if it is set")
	ntfyAuth = flag.String("ntfy-auth", envNtfyAuth, "Specify token for reserved topics")
	slackWebhookUrl = flag.String("slack-webhook", envSlackWebhookUrl, "Choose the slack webhook url to send messages to\nDefaults to the value of the SLACK_WEBHOOK_URL env var, if it is set")
	version := flag.Bool("v", false, "prints current ntfy-to-slack version")

	flag.Parse()

	if *version {
		println(VERSION)
		os.Exit(0)
	}

	client := &http.Client{}
	req, err := http.NewRequest("GET", "https://"+*ntfyDomain+"/"+*ntfyTopic+"/json", nil)
	if ntfyAuth != nil {
		req.Header.Add("Authorization", "Bearer "+*ntfyAuth)
	}

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("bot error: error on https attempt. verify network connectivity is OK. waiting 30 seconds before restarting.")
		time.Sleep(30 * time.Second)
		log.Fatal(err)
	} else if resp.StatusCode != http.StatusOK {
		sendToSlack("bot error: expected 200 OK from " + *ntfyDomain + ", instead: " + strconv.Itoa(resp.StatusCode) + ". waiting 30 seconds before restarting.")
		fmt.Printf("bot error: expected 200 OK from %s, instead: %s. waiting 30 seconds before restarting.", *ntfyDomain, strconv.Itoa(resp.StatusCode))
		time.Sleep(30 * time.Second)
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
