package main

import (
	slack "github.com/ashwanthkumar/slack-go-webhook"

	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

const VERSION = "v1 2022-10-26"
const NtfyServer = "ntfy.sh"
const NtfyTopic = "<ntfy-topic-goes-here>"
const SlackWebHookUrl = "<slack-webhook-url-goes-here>"

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
		Text: "(" + NtfyTopic + ") " + message,
	}

	err := slack.Send(SlackWebHookUrl, "", payload)
	if len(err) > 0 {
		fmt.Printf("error: %s\n", err)
	}
}

func main() {
	if len(os.Args[1:]) > 0 {
		if os.Args[1] == "-v" {
			println(VERSION)
			os.Exit(0)
		}
	}

	resp, err := http.Get("https://" + NtfyServer + "/" + NtfyTopic + "/json")
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
			fmt.Printf("%s: %s subscription established\n", timeT, NtfyServer)
			sendToSlack("bot restarted; " + NtfyServer +" subscription established")
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
