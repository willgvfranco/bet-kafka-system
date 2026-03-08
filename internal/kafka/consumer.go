package kafka

import (
	"context"
	"log/slog"
	"time"

	"github.com/neus/bet-kafka-system/internal/observability"
	kafkago "github.com/segmentio/kafka-go"
)

type MessageHandler func(ctx context.Context, msg kafkago.Message) error

type Consumer struct {
	reader  *kafkago.Reader
	handler MessageHandler
	logger  *slog.Logger
	topic   string
	groupID string
}

func NewConsumer(brokers, topic, groupID string, handler MessageHandler) *Consumer {
	reader := kafkago.NewReader(kafkago.ReaderConfig{
		Brokers:  []string{brokers},
		Topic:    topic,
		GroupID:  groupID,
		MinBytes: 1,
		MaxBytes: 10e6,
	})
	return &Consumer{
		reader:  reader,
		handler: handler,
		logger:  slog.With("topic", topic, "consumer_group", groupID),
		topic:   topic,
		groupID: groupID,
	}
}

func (c *Consumer) Run(ctx context.Context) error {
	c.logger.Info("consumer_started")
	for {
		select {
		case <-ctx.Done():
			c.logger.Info("consumer_stopped")
			return c.reader.Close()
		default:
		}

		msg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return c.reader.Close()
			}
			c.logger.Error("message_fetch_failed", "error", err)
			continue
		}

		start := time.Now()
		if err := c.handler(ctx, msg); err != nil {
			observability.KafkaProcessingErrors.WithLabelValues(c.topic, c.groupID).Inc()
			c.logger.Error("message_processing_failed",
				"error", err,
				"partition", msg.Partition,
				"offset", msg.Offset,
			)
			continue
		}
		duration := time.Since(start).Seconds()
		observability.KafkaMessagesConsumed.WithLabelValues(c.topic, c.groupID).Inc()
		observability.KafkaProcessingDuration.WithLabelValues(c.topic, c.groupID).Observe(duration)

		if err := c.reader.CommitMessages(ctx, msg); err != nil {
			c.logger.Error("offset_commit_failed", "error", err)
		}
	}
}
