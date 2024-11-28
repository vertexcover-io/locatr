package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/pebbe/zmq4"
	"github.com/playwright-community/playwright-go"
	"github.com/vertexcover-io/locatr"
)

type CTXKey string

var ErrorFatal = errors.New("fatal error")

var (
	config_key   CTXKey = "config"
	msg_chan_key CTXKey = "message_channel"
	rep_chan_key CTXKey = "reply_channel"
	err_chan_key CTXKey = "error_channel"
)

type Config struct {
	ModelName   string
	ApiKey      string
	IPCLocation string
}

type Message struct {
	Id          string `json:"id"`
	Url         string `json:"url"`
	Description string `json:"description"`
}

type MessageReply struct {
	Id      string `json:"id"`
	Url     string `json:"url"`
	Locator string `json:"locator"`
}

func run_model(ctx context.Context, msg Message, browser playwright.Browser) {
	err_chan := ctx.Value(err_chan_key).(chan<- error)

	log.Print("[INFO] Message request received")

	page, err := browser.NewPage()
	if err != nil {
		err_chan <- fmt.Errorf("could not create page: %w", err)
	}
	if _, err := page.Goto(msg.Url); err != nil {
		err_chan <- fmt.Errorf("could not navigate to requested page: %w", err)
	}
	time.Sleep(5 * time.Second) // wait for page to load
	log.Printf("[INFO] Page %s loaded", msg.Url)

	cfg := ctx.Value(config_key).(Config)

	llmClient, err := locatr.NewLlmClient(
		locatr.OpenAI, // (openai | anthropic),
		cfg.ModelName,
		cfg.ApiKey,
	)
	if err != nil {
		err_chan <- fmt.Errorf("could not create llm client: %w", err)
	}
	options := locatr.BaseLocatrOptions{UseCache: true, LogConfig: locatr.LogConfig{Level: locatr.Info}, LlmClient: llmClient}

	playWrightLocatr := locatr.NewPlaywrightLocatr(page, options)

	element, err := playWrightLocatr.GetLocatrStr(msg.Description)
	if err != nil {
		err_chan <- fmt.Errorf("could not get locator: %w", err)
	}

	rep_chan := ctx.Value(rep_chan_key).(chan<- MessageReply)
	reply := MessageReply{
		Id:      msg.Id,
		Url:     msg.Url,
		Locator: element,
	}
	rep_chan <- reply
}

func run_browser(ctx context.Context) {
	err_chan := ctx.Value(err_chan_key).(chan<- error)
	msg_chan := ctx.Value(msg_chan_key).(<-chan Message)

	pw, err := playwright.Run()
	if err != nil {
		err_chan <- fmt.Errorf("could not start playwright: %w: %w", err, ErrorFatal)
	}
	defer func() {
		if err = pw.Stop(); err != nil {
			// This is not needed. The only time we terminate playwright is
			// when terminating application.
			err_chan <- fmt.Errorf("failed to terminate playwright instance: %w: %w", err, ErrorFatal)
		}
	}()

	browser, err := pw.Chromium.Launch(
		playwright.BrowserTypeLaunchOptions{
			Headless: playwright.Bool(true),
		},
	)
	if err != nil {
		err_chan <- fmt.Errorf("could not launch browser: %w: %w", err, ErrorFatal)
	}
	defer browser.Close()
	log.Print("[INFO] Started Browser instance")

	log.Print("[INFO] Waiting for messages")
outer:
	for {
		select {
		case <-ctx.Done():
			break outer
		case msg := <-msg_chan:
			go run_model(ctx, msg, browser)
		}
	}

	log.Print("[INFO] Terminating Playwright instance")
}

func run_zmq(ctx context.Context) {
	cfg := ctx.Value(config_key).(Config)
	err_chan := ctx.Value(err_chan_key).(chan<- error)
	msg_chan := ctx.Value(msg_chan_key).(chan<- Message)

	zctx, err := zmq4.NewContext()
	if err != nil {
		err_chan <- fmt.Errorf("failed to start zeromq: %w: %w", err, ErrorFatal)
	}

	socket, err := zctx.NewSocket(zmq4.REP)
	if err != nil {
		err_chan <- fmt.Errorf("failed to open socket: %w: %w", err, ErrorFatal)
	}
	location := "ipc://" + cfg.IPCLocation
	if err = socket.Bind(location); err != nil {
		err_chan <- fmt.Errorf("failed to bind to ipc: %w: %w", err, ErrorFatal)
	}

	log.Printf("[INFO] Server is now listening at %s", location)

	go func() {
		reply_chan := ctx.Value(rep_chan_key).(<-chan MessageReply)

	outer:
		for {
			select {
			case <-ctx.Done():
				break outer
			case reply := <-reply_chan:
				data, err := json.Marshal(reply)
				if err != nil {
					err_chan <- fmt.Errorf("failed to marshal json: %w: %w", err, ErrorFatal)
				}
				log.Printf("[INFO] Sending message reply")
				_, err = socket.SendBytes(data, 0)
				if err != nil {
					err_chan <- fmt.Errorf("failed to send message: %w", err)
				}
				log.Printf("[INFO] Message Reply sent")
			}
		}
	}()

	// TODO: Figure out how to properly close out connection, while not missing any messages passed
	for {
		data, err := socket.RecvBytes(0)
		if err != nil {
			err_chan <- fmt.Errorf("failed to receive message: %w", err)

			var msg Message
			if err = json.Unmarshal(data, &msg); err != nil {
				err_chan <- fmt.Errorf("invalid message received, dropping: %w", err)
				continue
			}

			msg_chan <- msg

			log.Print("[INFO] Message received")
		}
	}
}

func validateConfig(cfg Config) error {
	if cfg.ModelName == "" {
		return errors.New("missing OpenAI Model name")
	}
	if cfg.ApiKey == "" {
		return errors.New("missing OpenAI API Key")
	}
	return nil
}

func main() {
	var cfg Config
	flag.StringVar(&cfg.ModelName, "model", "", "pass in OpenAI model name")
	flag.StringVar(&cfg.ApiKey, "api_key", "", "pass in OpenAI API key")
	flag.StringVar(&cfg.IPCLocation, "ipc", "/tmp/locator-ipc", "pass in where to create IPC file")

	flag.Parse()

	if err := validateConfig(cfg); err != nil {
		log.Fatalf("invalid config: %v", err)
	}

	error_channel := make(chan error)
	message_channel := make(chan Message)
	reply_channel := make(chan MessageReply)

	ctx, cancel := context.WithCancel(context.Background())
	ctx = context.WithValue(ctx, config_key, cfg)
	ctx = context.WithValue(ctx, err_chan_key, error_channel)
	ctx = context.WithValue(ctx, msg_chan_key, message_channel)
	ctx = context.WithValue(ctx, rep_chan_key, reply_channel)

	go run_browser(ctx)
	go run_zmq(ctx)

	for {
		err := <-error_channel
		if errors.Is(err, ErrorFatal) {
			cancel()
			log.Print("[INFO] Terminating application")
			log.Fatalf("[ERROR] Fatal error received: %v", err)
		}
		log.Printf("[WARN] error received: %v", err)
	}
}
