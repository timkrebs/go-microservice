package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/timkrebs/image-processor/internal/models"
)

// Consumer reads jobs from the Redis stream
type Consumer struct {
	client        *redis.Client
	streamName    string
	consumerGroup string
	consumerName  string
	pollTimeout   time.Duration
	logger        *slog.Logger
}

// ConsumerConfig holds consumer configuration
type ConsumerConfig struct {
	StreamName    string
	ConsumerGroup string
	ConsumerName  string
	PollTimeout   time.Duration
}

// NewConsumer creates a new queue consumer
func NewConsumer(client *redis.Client, cfg ConsumerConfig, logger *slog.Logger) *Consumer {
	return &Consumer{
		client:        client,
		streamName:    cfg.StreamName,
		consumerGroup: cfg.ConsumerGroup,
		consumerName:  cfg.ConsumerName,
		pollTimeout:   cfg.PollTimeout,
		logger:        logger,
	}
}

// EnsureGroup creates the consumer group if it doesn't exist
func (c *Consumer) EnsureGroup(ctx context.Context) error {
	err := c.client.XGroupCreateMkStream(ctx, c.streamName, c.consumerGroup, "0").Err()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		return fmt.Errorf("failed to create consumer group: %w", err)
	}
	return nil
}

// Message represents a message from the queue
type Message struct {
	ID   string
	Job  *models.JobMessage
	Data string
}

// Consume reads messages from the queue
func (c *Consumer) Consume(ctx context.Context) (*Message, error) {
	// First, try to read pending messages (messages that were read but not acknowledged)
	pendingMessages, err := c.client.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    c.consumerGroup,
		Consumer: c.consumerName,
		Streams:  []string{c.streamName, "0"},
		Count:    1,
		Block:    0, // Non-blocking for pending
	}).Result()
	if err != nil && err != redis.Nil {
		return nil, fmt.Errorf("failed to read pending messages: %w", err)
	}

	// Check if we got pending messages
	if len(pendingMessages) > 0 && len(pendingMessages[0].Messages) > 0 {
		return c.parseMessage(pendingMessages[0].Messages[0])
	}

	// No pending messages, read new messages
	streams, err := c.client.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    c.consumerGroup,
		Consumer: c.consumerName,
		Streams:  []string{c.streamName, ">"},
		Count:    1,
		Block:    c.pollTimeout,
	}).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // No messages available
		}
		return nil, fmt.Errorf("failed to read from stream: %w", err)
	}

	if len(streams) == 0 || len(streams[0].Messages) == 0 {
		return nil, nil
	}

	return c.parseMessage(streams[0].Messages[0])
}

func (c *Consumer) parseMessage(redisMsg redis.XMessage) (*Message, error) {
	data, ok := redisMsg.Values["data"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid message format: missing data field")
	}

	var jobMsg models.JobMessage
	if err := json.Unmarshal([]byte(data), &jobMsg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal job message: %w", err)
	}

	return &Message{
		ID:   redisMsg.ID,
		Job:  &jobMsg,
		Data: data,
	}, nil
}

// Acknowledge marks a message as processed
func (c *Consumer) Acknowledge(ctx context.Context, messageID string) error {
	_, err := c.client.XAck(ctx, c.streamName, c.consumerGroup, messageID).Result()
	if err != nil {
		return fmt.Errorf("failed to acknowledge message: %w", err)
	}
	return nil
}

// GetPendingCount returns the number of pending messages in the consumer group
func (c *Consumer) GetPendingCount(ctx context.Context) (int64, error) {
	pending, err := c.client.XPending(ctx, c.streamName, c.consumerGroup).Result()
	if err != nil {
		return 0, err
	}
	return pending.Count, nil
}
