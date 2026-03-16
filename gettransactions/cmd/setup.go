package main

import (
	"context"
	"fmt"

	"github.com/aws/aws-lambda-go/events"
	"github.com/awslabs/aws-lambda-go-api-proxy/httpadapter"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"draftea-challenge/gettransactions/internal/infra/handler"
	"draftea-challenge/gettransactions/internal/infra/repository/postgres"
	"draftea-challenge/gettransactions/internal/usecase"
)

func initTracer() (*sdktrace.TracerProvider, error) {
	exporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
	if err != nil {
		return nil, fmt.Errorf("creating stdout exporter: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
	)
	otel.SetTracerProvider(tp)

	return tp, nil
}

func setup() (func(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error), func(), error) {
	tp, err := initTracer()
	if err != nil {
		return nil, nil, fmt.Errorf("initializing tracer: %w", err)
	}

	db, err := postgres.NewConnection()
	if err != nil {
		return nil, nil, fmt.Errorf("connecting to database: %w", err)
	}

	walletRepo := postgres.NewWalletRepository(db)

	uc := usecase.NewGetTransactions(walletRepo)

	h := handler.New(uc)
	router := handler.NewRouter(h)

	adapter := httpadapter.New(router)

	sqlDB, _ := db.DB()
	cleanup := func() {
		tp.Shutdown(context.Background())
		if sqlDB != nil {
			sqlDB.Close()
		}
	}

	return adapter.ProxyWithContext, cleanup, nil
}
