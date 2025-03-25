package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"

	locatr "github.com/vertexcover-io/locatr/golang"
	appiumLocatr "github.com/vertexcover-io/locatr/golang/appium"
	"github.com/vertexcover-io/locatr/golang/llm"
	"github.com/vertexcover-io/locatr/golang/logger"
	"github.com/vertexcover-io/locatr/golang/tracing"
)

type Config struct {
	Tracing struct {
		Endpoint    string
		ServiceName string
		Insecure    bool
	}
}

func main() {
	ctx := context.Background()

	logger.Level.Set(slog.LevelDebug)

	cfg := Config{}
	cfg.Tracing.Endpoint = "localhost:4317"
	cfg.Tracing.ServiceName = "locator"
	cfg.Tracing.Insecure = true

	shutdown, err := tracing.SetupOtelSDK(
		context.Background(),
		tracing.WithEndpoint(cfg.Tracing.Endpoint),
		tracing.WithSVCName(cfg.Tracing.ServiceName),
		tracing.WithInsecure(cfg.Tracing.Insecure),
	)
	if err != nil {
		logger.Logger.Error(
			"Failed to setup Open Telemetry SDK",
			slog.String("error", err.Error()),
		)
		os.Exit(1)
	}
	defer func() {
		if sErr := shutdown(context.Background()); sErr != nil {
			err = errors.Join(err, sErr)
			logger.Logger.Error(
				"Error while shutting down Open Telemetry service",
				slog.String("error", err.Error()),
			)
		}
	}()

	llmClient, _ := llm.NewLlmClient(
		llm.OpenAI, // (openai | anthropic),
		os.Getenv("OPENAI_MODEL"),
		os.Getenv("OPENAI_KEY"),
	)
	bLocatr := locatr.BaseLocatrOptions{
		LlmClient: llmClient,
	}
	aLocatr, err := appiumLocatr.NewAppiumLocatr(
		ctx,
		"http://localhost:4723",
		"e82076df-7186-4e53-b7bb-a7a5ac43332a", bLocatr,
	)
	if err != nil {
		fmt.Println("failed creating appium locatr locatr", err)
		return
	}
	desc := "This EditText element for 'Email' serves as the second interactive entry point for user input in a linear sequence, positioned directly below an ImageView-containing LinearLayout which implies a visual hierarchy. It follows a specific order that leads to it being the first editable text input provided in the context, indicating its primary role in user authentication workflows."
	l, err := aLocatr.GetLocatrStr(ctx, desc)
	if err != nil {
		fmt.Println("error getting locatr", err)
	}
	fmt.Println(l)
}
