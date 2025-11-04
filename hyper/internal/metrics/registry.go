package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

// Global metric collectors
var (
	// WebSocket Metrics
	WebSocketConnectionsTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "chat_websocket_connections_total",
		Help: "Total number of WebSocket connections established",
	})

	WebSocketActiveConnections = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "chat_websocket_active_connections",
		Help: "Current number of active WebSocket connections",
	})

	WebSocketMessagesSent = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "chat_websocket_messages_sent_total",
		Help: "Total number of WebSocket messages sent to clients",
	})

	WebSocketMessagesReceived = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "chat_websocket_messages_received_total",
		Help: "Total number of WebSocket messages received from clients",
	})

	WebSocketMessageSize = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "chat_websocket_message_size_bytes",
		Help:    "Size distribution of WebSocket messages in bytes",
		Buckets: []float64{100, 500, 1024, 5120, 10240, 51200, 102400, 512000, 1048576}, // 100B to 1MB
	})

	WebSocketConnectionDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "chat_websocket_connection_duration_seconds",
		Help:    "Duration distribution of WebSocket connections",
		Buckets: prometheus.ExponentialBuckets(1, 2, 12), // 1s to ~68 minutes
	})

	WebSocketErrors = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "chat_websocket_errors_total",
		Help: "Total number of WebSocket errors by type",
	}, []string{"error_type"})

	// Message Validation Metrics (from our new feature!)
	MessageValidationRejections = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "chat_message_validation_rejections_total",
		Help: "Total number of message validation rejections by layer",
	}, []string{"layer"}) // layer: websocket, content, service

	MessageSizeExceeded = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "chat_message_size_exceeded_total",
		Help: "Total number of messages exceeding size limits by type",
	}, []string{"type"}) // type: content, tool_result, stream

	AIResponseTruncations = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "chat_ai_response_truncations_total",
		Help: "Total number of AI responses truncated due to size limits",
	})

	// Chat Session Metrics
	ChatSessionsCreated = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "chat_sessions_created_total",
		Help: "Total number of chat sessions created",
	})

	ChatSessionDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "chat_session_duration_seconds",
		Help:    "Duration distribution of chat sessions",
		Buckets: prometheus.ExponentialBuckets(60, 2, 10), // 1 min to ~17 hours
	})

	// Chat Message Metrics
	ChatMessagesSaved = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "chat_messages_saved_total",
		Help: "Total number of chat messages saved by role",
	}, []string{"role"}) // role: user, assistant

	ChatMessageSaveDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "chat_message_save_duration_seconds",
		Help:    "Duration distribution of message save operations",
		Buckets: prometheus.ExponentialBuckets(0.001, 2, 10), // 1ms to ~1 second
	})

	// AI Streaming Metrics
	AIStreamTokens = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "chat_ai_stream_tokens_total",
		Help: "Total number of AI streaming tokens generated",
	})

	AIStreamDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "chat_ai_stream_duration_seconds",
		Help:    "Duration distribution of AI streaming responses",
		Buckets: prometheus.ExponentialBuckets(1, 2, 8), // 1s to ~4 minutes
	})

	AIRequestsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "chat_ai_requests_total",
		Help: "Total number of AI API requests by provider and model",
	}, []string{"provider", "model"})

	AIRequestDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "chat_ai_request_duration_seconds",
		Help:    "Duration distribution of AI API requests",
		Buckets: prometheus.ExponentialBuckets(0.1, 2, 10), // 100ms to ~100s
	}, []string{"provider", "model"})

	AITokensConsumed = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "chat_ai_tokens_consumed_total",
		Help: "Total number of AI tokens consumed by model",
	}, []string{"model", "type"}) // type: input, output

	AIErrors = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "chat_ai_errors_total",
		Help: "Total number of AI API errors by type",
	}, []string{"error_type"})

	// HTTP Metrics
	HTTPRequestsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total number of HTTP requests by method, endpoint, and status",
	}, []string{"method", "endpoint", "status"})

	HTTPRequestDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "Duration distribution of HTTP requests",
		Buckets: prometheus.ExponentialBuckets(0.001, 2, 12), // 1ms to ~4s
	}, []string{"method", "endpoint"})

	HTTPRequestSize = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "http_request_size_bytes",
		Help:    "Size distribution of HTTP request bodies",
		Buckets: []float64{100, 1024, 10240, 102400, 1048576}, // 100B to 1MB
	})

	HTTPResponseSize = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "http_response_size_bytes",
		Help:    "Size distribution of HTTP response bodies",
		Buckets: []float64{100, 1024, 10240, 102400, 1048576, 10485760}, // 100B to 10MB
	})

	// Database Metrics
	MongoDBQueryDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "mongodb_query_duration_seconds",
		Help:    "Duration distribution of MongoDB queries by operation",
		Buckets: prometheus.ExponentialBuckets(0.001, 2, 10), // 1ms to ~1s
	}, []string{"operation", "collection"})

	MongoDBQueryErrors = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "mongodb_query_errors_total",
		Help: "Total number of MongoDB query errors",
	}, []string{"operation", "collection"})

	MongoDBTransactionDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "mongodb_transaction_duration_seconds",
		Help:    "Duration distribution of MongoDB transactions",
		Buckets: prometheus.ExponentialBuckets(0.01, 2, 8), // 10ms to ~1.2s
	})
)

// Registry is the Prometheus registry for all metrics
var Registry *prometheus.Registry

// init initializes the metrics registry and registers all collectors
func init() {
	Registry = prometheus.NewRegistry()

	// Register Go runtime metrics (goroutines, memory, GC, etc.)
	Registry.MustRegister(collectors.NewGoCollector())
	Registry.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))

	// Register WebSocket metrics
	Registry.MustRegister(WebSocketConnectionsTotal)
	Registry.MustRegister(WebSocketActiveConnections)
	Registry.MustRegister(WebSocketMessagesSent)
	Registry.MustRegister(WebSocketMessagesReceived)
	Registry.MustRegister(WebSocketMessageSize)
	Registry.MustRegister(WebSocketConnectionDuration)
	Registry.MustRegister(WebSocketErrors)

	// Register validation metrics
	Registry.MustRegister(MessageValidationRejections)
	Registry.MustRegister(MessageSizeExceeded)
	Registry.MustRegister(AIResponseTruncations)

	// Register chat session metrics
	Registry.MustRegister(ChatSessionsCreated)
	Registry.MustRegister(ChatSessionDuration)

	// Register chat message metrics
	Registry.MustRegister(ChatMessagesSaved)
	Registry.MustRegister(ChatMessageSaveDuration)

	// Register AI metrics
	Registry.MustRegister(AIStreamTokens)
	Registry.MustRegister(AIStreamDuration)
	Registry.MustRegister(AIRequestsTotal)
	Registry.MustRegister(AIRequestDuration)
	Registry.MustRegister(AITokensConsumed)
	Registry.MustRegister(AIErrors)

	// Register HTTP metrics
	Registry.MustRegister(HTTPRequestsTotal)
	Registry.MustRegister(HTTPRequestDuration)
	Registry.MustRegister(HTTPRequestSize)
	Registry.MustRegister(HTTPResponseSize)

	// Register database metrics
	Registry.MustRegister(MongoDBQueryDuration)
	Registry.MustRegister(MongoDBQueryErrors)
	Registry.MustRegister(MongoDBTransactionDuration)
}

// Helper functions for common metric operations

// RecordWebSocketConnection increments connection counter and active gauge
func RecordWebSocketConnection() {
	WebSocketConnectionsTotal.Inc()
	WebSocketActiveConnections.Inc()
}

// RecordWebSocketDisconnection decrements active connections gauge
func RecordWebSocketDisconnection() {
	WebSocketActiveConnections.Dec()
}

// RecordValidationRejection records a message validation rejection
func RecordValidationRejection(layer string) {
	MessageValidationRejections.WithLabelValues(layer).Inc()
}

// RecordMessageSizeExceeded records a message size limit violation
func RecordMessageSizeExceeded(messageType string) {
	MessageSizeExceeded.WithLabelValues(messageType).Inc()
}

// RecordChatMessage records a saved chat message
func RecordChatMessage(role string, duration float64) {
	ChatMessagesSaved.WithLabelValues(role).Inc()
	ChatMessageSaveDuration.Observe(duration)
}

// RecordAIRequest records an AI API request
func RecordAIRequest(provider, model string, duration float64) {
	AIRequestsTotal.WithLabelValues(provider, model).Inc()
	AIRequestDuration.WithLabelValues(provider, model).Observe(duration)
}

// RecordAITokens records AI token consumption
func RecordAITokens(model, tokenType string, count int) {
	AITokensConsumed.WithLabelValues(model, tokenType).Add(float64(count))
}

// RecordHTTPRequest records an HTTP request
func RecordHTTPRequest(method, endpoint, status string, duration float64) {
	HTTPRequestsTotal.WithLabelValues(method, endpoint, status).Inc()
	HTTPRequestDuration.WithLabelValues(method, endpoint).Observe(duration)
}

// RecordMongoDBQuery records a MongoDB query
func RecordMongoDBQuery(operation, collection string, duration float64, err error) {
	MongoDBQueryDuration.WithLabelValues(operation, collection).Observe(duration)
	if err != nil {
		MongoDBQueryErrors.WithLabelValues(operation, collection).Inc()
	}
}
