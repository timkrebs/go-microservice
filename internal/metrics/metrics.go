package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// HTTPMetrics holds HTTP-related Prometheus metrics
type HTTPMetrics struct {
	RequestDuration  *prometheus.HistogramVec
	RequestsTotal    *prometheus.CounterVec
	RequestsInFlight prometheus.Gauge
}

// NewHTTPMetrics creates HTTP metrics collectors
func NewHTTPMetrics(namespace string) *HTTPMetrics {
	return &HTTPMetrics{
		RequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "http_request_duration_seconds",
				Help:      "HTTP request duration in seconds",
				Buckets:   []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
			},
			[]string{"method", "path", "status"},
		),
		RequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "http_requests_total",
				Help:      "Total number of HTTP requests",
			},
			[]string{"method", "path", "status"},
		),
		RequestsInFlight: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "http_requests_in_flight",
				Help:      "Current number of HTTP requests being processed",
			},
		),
	}
}

// JobMetrics holds job processing Prometheus metrics
type JobMetrics struct {
	ProcessingDuration *prometheus.HistogramVec
	JobsTotal          *prometheus.CounterVec
	JobsActive         *prometheus.GaugeVec
	OperationDuration  *prometheus.HistogramVec
}

// NewJobMetrics creates job processing metrics collectors
func NewJobMetrics(namespace string) *JobMetrics {
	return &JobMetrics{
		ProcessingDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "job_processing_duration_seconds",
				Help:      "Job processing duration in seconds",
				Buckets:   []float64{.1, .5, 1, 2.5, 5, 10, 30, 60, 120, 300},
			},
			[]string{"status"},
		),
		JobsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "jobs_total",
				Help:      "Total number of jobs processed",
			},
			[]string{"status"},
		),
		JobsActive: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "jobs_active",
				Help:      "Number of currently active jobs",
			},
			[]string{"status"},
		),
		OperationDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "job_operation_duration_seconds",
				Help:      "Individual operation processing duration in seconds",
				Buckets:   []float64{.01, .05, .1, .25, .5, 1, 2.5, 5, 10},
			},
			[]string{"operation"},
		),
	}
}

// QueueMetrics holds queue-related Prometheus metrics
type QueueMetrics struct {
	Depth            prometheus.Gauge
	MessagesProduced prometheus.Counter
	MessagesConsumed prometheus.Counter
	MessagesFailed   prometheus.Counter
	ConsumeDuration  prometheus.Histogram
}

// NewQueueMetrics creates queue metrics collectors
func NewQueueMetrics(namespace string) *QueueMetrics {
	return &QueueMetrics{
		Depth: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "queue_depth",
				Help:      "Current number of messages in the queue",
			},
		),
		MessagesProduced: promauto.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "queue_messages_produced_total",
				Help:      "Total number of messages produced to the queue",
			},
		),
		MessagesConsumed: promauto.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "queue_messages_consumed_total",
				Help:      "Total number of messages consumed from the queue",
			},
		),
		MessagesFailed: promauto.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "queue_messages_failed_total",
				Help:      "Total number of messages that failed processing",
			},
		),
		ConsumeDuration: promauto.NewHistogram(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "queue_consume_duration_seconds",
				Help:      "Time spent consuming messages from the queue",
				Buckets:   []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1},
			},
		),
	}
}

// StorageMetrics holds storage operation Prometheus metrics
type StorageMetrics struct {
	OperationDuration *prometheus.HistogramVec
	OperationsTotal   *prometheus.CounterVec
	BytesTransferred  *prometheus.CounterVec
}

// NewStorageMetrics creates storage metrics collectors
func NewStorageMetrics(namespace string) *StorageMetrics {
	return &StorageMetrics{
		OperationDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "storage_operation_duration_seconds",
				Help:      "Storage operation duration in seconds",
				Buckets:   []float64{.01, .05, .1, .25, .5, 1, 2.5, 5, 10, 30},
			},
			[]string{"operation", "status"},
		),
		OperationsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "storage_operations_total",
				Help:      "Total number of storage operations",
			},
			[]string{"operation", "status"},
		),
		BytesTransferred: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "storage_bytes_transferred_total",
				Help:      "Total number of bytes transferred to/from storage",
			},
			[]string{"operation"},
		),
	}
}

// DatabaseMetrics holds database operation Prometheus metrics
type DatabaseMetrics struct {
	QueryDuration     *prometheus.HistogramVec
	QueriesTotal      *prometheus.CounterVec
	ConnectionsActive prometheus.Gauge
}

// NewDatabaseMetrics creates database metrics collectors
func NewDatabaseMetrics(namespace string) *DatabaseMetrics {
	return &DatabaseMetrics{
		QueryDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "database_query_duration_seconds",
				Help:      "Database query duration in seconds",
				Buckets:   []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5},
			},
			[]string{"operation", "status"},
		),
		QueriesTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "database_queries_total",
				Help:      "Total number of database queries",
			},
			[]string{"operation", "status"},
		),
		ConnectionsActive: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "database_connections_active",
				Help:      "Number of active database connections",
			},
		),
	}
}

// RecordDuration helper to record operation duration
func RecordDuration(start time.Time, histogram prometheus.Observer) {
	duration := time.Since(start).Seconds()
	histogram.Observe(duration)
}
