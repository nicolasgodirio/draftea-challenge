package main

import (
	"log/slog"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	h, cleanup, err := setup()
	if err != nil {
		slog.Error("failed to setup service", "error", err)
		os.Exit(1)
	}
	defer cleanup()

	lambda.Start(h)
}
