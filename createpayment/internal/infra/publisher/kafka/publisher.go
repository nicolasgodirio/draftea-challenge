package kafka

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	"draftea-challenge/createpayment/internal/domain"
)

const topic = "payment.created"

type PaymentPublisher struct{}

func NewPaymentPublisher() *PaymentPublisher {
	return &PaymentPublisher{}
}

func (p *PaymentPublisher) Publish(ctx context.Context, payment *domain.Payment) error {
	ctx, span := otel.Tracer("createpayment").Start(ctx, "KafkaPublisher.Publish")
	defer span.End()

	span.SetAttributes(
		attribute.String("topic", topic),
		attribute.String("payment_id", payment.ID),
		attribute.String("user_id", payment.UserID),
	)

	slog.InfoContext(ctx, "publishing payment event to kafka",
		"topic", topic,
		"payment_id", payment.ID,
		"user_id", payment.UserID,
		"amount", payment.Amount,
	)

	// TODO: Replace with actual Kafka producer (e.g. confluent-kafka-go or segmentio/kafka-go)
	// msg := &kafka.Message{
	//     Topic: topic,
	//     Key:   []byte(payment.UserID),
	//     Value: payloadBytes,
	// }
	// return producer.Produce(ctx, msg)

	span.SetStatus(codes.Ok, "published")
	return nil
}
