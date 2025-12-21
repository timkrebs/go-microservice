package queue

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/redis/go-redis/v9"

	"github.com/timkrebs/image-processor/internal/models"
)

// Producer publishes jobs to the Redis stream
type Producer struct {
	client     *redis.Client
	streamName string
}

// NewProducer creates a new queue producer
func NewProducer(client *redis.Client, streamName string) *Producer {
	return &Producer{
		client:     client,
		streamName: streamName,
	}
}

// Enqueue adds a job to the queue
func (p *Producer) Enqueue(ctx context.Context, msg *models.JobMessage) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal job message: %w", err)
	}

	_, err = p.client.XAdd(ctx, &redis.XAddArgs{
		Stream: p.streamName,
		Values: map[string]interface{}{
			"data": string(data),
		},
	}).Result()
	if err != nil {
		return fmt.Errorf("failed to add to stream: %w", err)
	}

	return nil
}

// GetStreamLength returns the current length of the stream
func (p *Producer) GetStreamLength(ctx context.Context) (int64, error) {
	return p.client.XLen(ctx, p.streamName).Result()
}

// GetStats returns queue statistics
func (p *Producer) GetStats(ctx context.Context, consumerGroup string) (*models.QueueStats, error) {
	stats := &models.QueueStats{}

	// Get stream length
	length, err := p.client.XLen(ctx, p.streamName).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get stream length: %w", err)
	}
	stats.StreamLength = length

	// Get pending messages info
	pending, err := p.client.XPending(ctx, p.streamName, consumerGroup).Result()
	if err != nil {
		// Group might not exist yet
		stats.PendingMessages = 0
		stats.ConsumerCount = 0
	} else {
		stats.PendingMessages = pending.Count
		stats.ConsumerCount = int64(len(pending.Consumers))
	}

	return stats, nil
}
