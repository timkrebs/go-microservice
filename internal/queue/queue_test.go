package queue

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/timkrebs/image-processor/internal/models"
)

// getTestRedisClient creates a Redis client for testing
// Returns nil if Redis is not available (for CI environments without Redis)
func getTestRedisClient(t *testing.T) *redis.Client {
	t.Helper()

	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "localhost:6379"
	}

	client := redis.NewClient(&redis.Options{
		Addr: addr,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		t.Skipf("Redis not available at %s: %v", addr, err)
		return nil
	}

	return client
}

// cleanupStream deletes the test stream
func cleanupStream(t *testing.T, client *redis.Client, streamName string) {
	t.Helper()
	ctx := context.Background()
	client.Del(ctx, streamName)
}

func TestNewProducer(t *testing.T) {
	client := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	defer client.Close()

	producer := NewProducer(client, "test-stream")
	if producer == nil {
		t.Fatal("NewProducer() returned nil")
	}
	if producer.streamName != "test-stream" {
		t.Errorf("streamName = %q, want test-stream", producer.streamName)
	}
}

func TestNewConsumer(t *testing.T) {
	client := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	defer client.Close()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cfg := ConsumerConfig{
		StreamName:    "test-stream",
		ConsumerGroup: "test-group",
		ConsumerName:  "test-consumer",
		PollTimeout:   5 * time.Second,
	}

	consumer := NewConsumer(client, cfg, logger)
	if consumer == nil {
		t.Fatal("NewConsumer() returned nil")
	}
	if consumer.streamName != "test-stream" {
		t.Errorf("streamName = %q, want test-stream", consumer.streamName)
	}
	if consumer.consumerGroup != "test-group" {
		t.Errorf("consumerGroup = %q, want test-group", consumer.consumerGroup)
	}
	if consumer.consumerName != "test-consumer" {
		t.Errorf("consumerName = %q, want test-consumer", consumer.consumerName)
	}
	if consumer.pollTimeout != 5*time.Second {
		t.Errorf("pollTimeout = %v, want 5s", consumer.pollTimeout)
	}
}

func TestProducer_Enqueue(t *testing.T) {
	client := getTestRedisClient(t)
	if client == nil {
		return
	}
	defer client.Close()

	streamName := "test-enqueue-" + uuid.New().String()[:8]
	defer cleanupStream(t, client, streamName)

	producer := NewProducer(client, streamName)

	msg := &models.JobMessage{
		JobID: uuid.New(),
		Operations: []models.Operation{
			{Operation: models.OperationResize, Parameters: map[string]interface{}{"width": 100}},
		},
	}

	err := producer.Enqueue(context.Background(), msg)
	if err != nil {
		t.Fatalf("Enqueue() error = %v", err)
	}

	// Verify message was added
	length, err := producer.GetStreamLength(context.Background())
	if err != nil {
		t.Fatalf("GetStreamLength() error = %v", err)
	}
	if length != 1 {
		t.Errorf("StreamLength = %d, want 1", length)
	}
}

func TestProducer_GetStreamLength(t *testing.T) {
	client := getTestRedisClient(t)
	if client == nil {
		return
	}
	defer client.Close()

	streamName := "test-length-" + uuid.New().String()[:8]
	defer cleanupStream(t, client, streamName)

	producer := NewProducer(client, streamName)

	// Empty stream
	length, err := producer.GetStreamLength(context.Background())
	if err != nil {
		t.Fatalf("GetStreamLength() error = %v", err)
	}
	if length != 0 {
		t.Errorf("Empty stream length = %d, want 0", length)
	}

	// Add messages
	for i := 0; i < 5; i++ {
		msg := &models.JobMessage{JobID: uuid.New()}
		if err := producer.Enqueue(context.Background(), msg); err != nil {
			t.Fatalf("Enqueue() error = %v", err)
		}
	}

	length, err = producer.GetStreamLength(context.Background())
	if err != nil {
		t.Fatalf("GetStreamLength() error = %v", err)
	}
	if length != 5 {
		t.Errorf("StreamLength = %d, want 5", length)
	}
}

func TestProducer_GetStats(t *testing.T) {
	client := getTestRedisClient(t)
	if client == nil {
		return
	}
	defer client.Close()

	streamName := "test-stats-" + uuid.New().String()[:8]
	consumerGroup := "test-group"
	defer cleanupStream(t, client, streamName)

	producer := NewProducer(client, streamName)

	// Add a message first
	msg := &models.JobMessage{JobID: uuid.New()}
	if err := producer.Enqueue(context.Background(), msg); err != nil {
		t.Fatalf("Enqueue() error = %v", err)
	}

	// Get stats (group doesn't exist yet)
	stats, err := producer.GetStats(context.Background(), consumerGroup)
	if err != nil {
		t.Fatalf("GetStats() error = %v", err)
	}

	if stats.StreamLength != 1 {
		t.Errorf("StreamLength = %d, want 1", stats.StreamLength)
	}
	// Consumer group doesn't exist, so pending should be 0
	if stats.PendingMessages != 0 {
		t.Errorf("PendingMessages = %d, want 0", stats.PendingMessages)
	}
}

func TestConsumer_EnsureGroup(t *testing.T) {
	client := getTestRedisClient(t)
	if client == nil {
		return
	}
	defer client.Close()

	streamName := "test-ensure-group-" + uuid.New().String()[:8]
	defer cleanupStream(t, client, streamName)

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cfg := ConsumerConfig{
		StreamName:    streamName,
		ConsumerGroup: "test-group",
		ConsumerName:  "test-consumer",
		PollTimeout:   time.Second,
	}

	consumer := NewConsumer(client, cfg, logger)

	// First call should create the group
	err := consumer.EnsureGroup(context.Background())
	if err != nil {
		t.Fatalf("EnsureGroup() first call error = %v", err)
	}

	// Second call should not error (group already exists)
	err = consumer.EnsureGroup(context.Background())
	if err != nil {
		t.Fatalf("EnsureGroup() second call error = %v", err)
	}
}

func TestConsumer_Consume_NoMessages(t *testing.T) {
	client := getTestRedisClient(t)
	if client == nil {
		return
	}
	defer client.Close()

	streamName := "test-consume-empty-" + uuid.New().String()[:8]
	defer cleanupStream(t, client, streamName)

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cfg := ConsumerConfig{
		StreamName:    streamName,
		ConsumerGroup: "test-group",
		ConsumerName:  "test-consumer",
		PollTimeout:   100 * time.Millisecond, // Short timeout for test
	}

	consumer := NewConsumer(client, cfg, logger)

	// Create the group first
	if err := consumer.EnsureGroup(context.Background()); err != nil {
		t.Fatalf("EnsureGroup() error = %v", err)
	}

	// Try to consume from empty stream
	msg, err := consumer.Consume(context.Background())
	if err != nil {
		t.Fatalf("Consume() error = %v", err)
	}
	if msg != nil {
		t.Errorf("Expected nil message from empty stream, got %+v", msg)
	}
}

func TestConsumer_Consume_WithMessage(t *testing.T) {
	client := getTestRedisClient(t)
	if client == nil {
		return
	}
	defer client.Close()

	streamName := "test-consume-msg-" + uuid.New().String()[:8]
	defer cleanupStream(t, client, streamName)

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Create producer and add message
	producer := NewProducer(client, streamName)
	expectedJobID := uuid.New()
	jobMsg := &models.JobMessage{
		JobID: expectedJobID,
		Operations: []models.Operation{
			{Operation: models.OperationBlur, Parameters: map[string]interface{}{"sigma": 2.0}},
		},
	}
	if err := producer.Enqueue(context.Background(), jobMsg); err != nil {
		t.Fatalf("Enqueue() error = %v", err)
	}

	// Create consumer and consume
	cfg := ConsumerConfig{
		StreamName:    streamName,
		ConsumerGroup: "test-group",
		ConsumerName:  "test-consumer",
		PollTimeout:   time.Second,
	}
	consumer := NewConsumer(client, cfg, logger)

	if err := consumer.EnsureGroup(context.Background()); err != nil {
		t.Fatalf("EnsureGroup() error = %v", err)
	}

	msg, err := consumer.Consume(context.Background())
	if err != nil {
		t.Fatalf("Consume() error = %v", err)
	}

	if msg == nil {
		t.Fatal("Expected message, got nil")
	}
	if msg.ID == "" {
		t.Error("Message ID should not be empty")
	}
	if msg.Job == nil {
		t.Fatal("Message Job should not be nil")
	}
	if msg.Job.JobID != expectedJobID {
		t.Errorf("JobID = %v, want %v", msg.Job.JobID, expectedJobID)
	}
	if len(msg.Job.Operations) != 1 {
		t.Errorf("len(Operations) = %d, want 1", len(msg.Job.Operations))
	}
}

func TestConsumer_Acknowledge(t *testing.T) {
	client := getTestRedisClient(t)
	if client == nil {
		return
	}
	defer client.Close()

	streamName := "test-ack-" + uuid.New().String()[:8]
	defer cleanupStream(t, client, streamName)

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Create producer and add message
	producer := NewProducer(client, streamName)
	jobMsg := &models.JobMessage{JobID: uuid.New()}
	if err := producer.Enqueue(context.Background(), jobMsg); err != nil {
		t.Fatalf("Enqueue() error = %v", err)
	}

	// Create consumer and consume
	cfg := ConsumerConfig{
		StreamName:    streamName,
		ConsumerGroup: "test-group",
		ConsumerName:  "test-consumer",
		PollTimeout:   time.Second,
	}
	consumer := NewConsumer(client, cfg, logger)

	if err := consumer.EnsureGroup(context.Background()); err != nil {
		t.Fatalf("EnsureGroup() error = %v", err)
	}

	msg, err := consumer.Consume(context.Background())
	if err != nil {
		t.Fatalf("Consume() error = %v", err)
	}
	if msg == nil {
		t.Fatal("Expected message, got nil")
	}

	// Acknowledge the message
	err = consumer.Acknowledge(context.Background(), msg.ID)
	if err != nil {
		t.Fatalf("Acknowledge() error = %v", err)
	}

	// Check pending count is 0
	pending, err := consumer.GetPendingCount(context.Background())
	if err != nil {
		t.Fatalf("GetPendingCount() error = %v", err)
	}
	if pending != 0 {
		t.Errorf("PendingCount = %d, want 0 after ack", pending)
	}
}

func TestConsumer_GetPendingCount(t *testing.T) {
	client := getTestRedisClient(t)
	if client == nil {
		return
	}
	defer client.Close()

	streamName := "test-pending-" + uuid.New().String()[:8]
	defer cleanupStream(t, client, streamName)

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Create producer and add messages
	producer := NewProducer(client, streamName)
	for i := 0; i < 3; i++ {
		jobMsg := &models.JobMessage{JobID: uuid.New()}
		if err := producer.Enqueue(context.Background(), jobMsg); err != nil {
			t.Fatalf("Enqueue() error = %v", err)
		}
	}

	// Create consumer
	cfg := ConsumerConfig{
		StreamName:    streamName,
		ConsumerGroup: "test-group",
		ConsumerName:  "test-consumer",
		PollTimeout:   time.Second,
	}
	consumer := NewConsumer(client, cfg, logger)

	if err := consumer.EnsureGroup(context.Background()); err != nil {
		t.Fatalf("EnsureGroup() error = %v", err)
	}

	// Consume 2 messages (leaving them pending)
	consumedCount := 0
	for i := 0; i < 2; i++ {
		msg, err := consumer.Consume(context.Background())
		if err != nil {
			t.Fatalf("Consume() error = %v", err)
		}
		if msg != nil {
			consumedCount++
		}
	}

	// Check pending count - should match what we consumed
	pending, err := consumer.GetPendingCount(context.Background())
	if err != nil {
		t.Fatalf("GetPendingCount() error = %v", err)
	}
	if pending != int64(consumedCount) {
		t.Errorf("PendingCount = %d, want %d", pending, consumedCount)
	}
}

func TestMessage_Fields(t *testing.T) {
	jobMsg := &models.JobMessage{
		JobID: uuid.New(),
		Operations: []models.Operation{
			{Operation: models.OperationGrayscale},
		},
	}

	data, _ := json.Marshal(jobMsg)

	msg := &Message{
		ID:   "1234567890-0",
		Job:  jobMsg,
		Data: string(data),
	}

	if msg.ID != "1234567890-0" {
		t.Errorf("ID = %q, want 1234567890-0", msg.ID)
	}
	if msg.Job == nil {
		t.Error("Job should not be nil")
	}
	if msg.Data == "" {
		t.Error("Data should not be empty")
	}
}

func TestConsumerConfig_Fields(t *testing.T) {
	cfg := ConsumerConfig{
		StreamName:    "my-stream",
		ConsumerGroup: "my-group",
		ConsumerName:  "worker-1",
		PollTimeout:   10 * time.Second,
	}

	if cfg.StreamName != "my-stream" {
		t.Errorf("StreamName = %q, want my-stream", cfg.StreamName)
	}
	if cfg.ConsumerGroup != "my-group" {
		t.Errorf("ConsumerGroup = %q, want my-group", cfg.ConsumerGroup)
	}
	if cfg.ConsumerName != "worker-1" {
		t.Errorf("ConsumerName = %q, want worker-1", cfg.ConsumerName)
	}
	if cfg.PollTimeout != 10*time.Second {
		t.Errorf("PollTimeout = %v, want 10s", cfg.PollTimeout)
	}
}

func TestProducerConsumer_Integration(t *testing.T) {
	client := getTestRedisClient(t)
	if client == nil {
		return
	}
	defer client.Close()

	streamName := "test-integration-" + uuid.New().String()[:8]
	defer cleanupStream(t, client, streamName)

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Create producer
	producer := NewProducer(client, streamName)

	// Create consumer
	cfg := ConsumerConfig{
		StreamName:    streamName,
		ConsumerGroup: "test-group",
		ConsumerName:  "test-consumer",
		PollTimeout:   time.Second,
	}
	consumer := NewConsumer(client, cfg, logger)

	if err := consumer.EnsureGroup(context.Background()); err != nil {
		t.Fatalf("EnsureGroup() error = %v", err)
	}

	// Produce multiple messages
	jobIDs := make([]uuid.UUID, 5)
	for i := 0; i < 5; i++ {
		jobIDs[i] = uuid.New()
		jobMsg := &models.JobMessage{
			JobID: jobIDs[i],
			Operations: []models.Operation{
				{Operation: models.OperationResize, Parameters: map[string]interface{}{"width": i * 100}},
			},
		}
		if err := producer.Enqueue(context.Background(), jobMsg); err != nil {
			t.Fatalf("Enqueue() error = %v", err)
		}
	}

	// Verify stream length
	length, err := producer.GetStreamLength(context.Background())
	if err != nil {
		t.Fatalf("GetStreamLength() error = %v", err)
	}
	if length != 5 {
		t.Errorf("StreamLength = %d, want 5", length)
	}

	// Consume all messages
	receivedIDs := make(map[uuid.UUID]bool)
	for i := 0; i < 5; i++ {
		msg, err := consumer.Consume(context.Background())
		if err != nil {
			t.Fatalf("Consume() error = %v", err)
		}
		if msg == nil {
			t.Fatalf("Expected message %d, got nil", i)
		}
		receivedIDs[msg.Job.JobID] = true

		// Acknowledge
		if err := consumer.Acknowledge(context.Background(), msg.ID); err != nil {
			t.Fatalf("Acknowledge() error = %v", err)
		}
	}

	// Verify all job IDs were received
	for _, id := range jobIDs {
		if !receivedIDs[id] {
			t.Errorf("Job ID %v was not received", id)
		}
	}

	// Verify no pending messages
	pending, err := consumer.GetPendingCount(context.Background())
	if err != nil {
		t.Fatalf("GetPendingCount() error = %v", err)
	}
	if pending != 0 {
		t.Errorf("PendingCount = %d, want 0", pending)
	}
}
