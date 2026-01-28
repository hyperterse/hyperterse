package connectors

import (
	"context"
	"fmt"
	"strings"

	"github.com/hyperterse/hyperterse/core/logger"
	"github.com/hyperterse/hyperterse/core/proto/hyperterse"
	"github.com/redis/go-redis/v9"
)

// RedisConnector implements the Connector interface for Redis
type RedisConnector struct {
	client *redis.Client
}

// NewRedisConnector creates a new Redis connector
func NewRedisConnector(connectionString string, options *hyperterse.AdapterOptions) (*RedisConnector, error) {
	log := logger.New("connector:redis")
	log.Debugf("Opening Redis connection")

	// Parse connection string (format: redis://user:password@host:port/db)
	opt, err := redis.ParseURL(connectionString)
	if err != nil {
		log.Errorf("Failed to parse Redis connection string: %v", err)
		return nil, fmt.Errorf("failed to parse redis connection string: %w", err)
	}

	// Apply options directly if provided
	// The Redis client will use the options from the parsed URL,
	// and any additional options can be applied here if needed
	// For now, we pass the options object directly without mapping
	_ = options // options can be used directly if needed in the future

	client := redis.NewClient(opt)

	// Test the connection
	log.Debugf("Testing connection with ping")
	if err := client.Ping(context.Background()).Err(); err != nil {
		client.Close()
		log.Errorf("Failed to ping Redis: %v", err)
		return nil, fmt.Errorf("failed to ping redis: %w", err)
	}

	log.Debugf("Redis connection opened successfully")
	return &RedisConnector{
		client: client,
	}, nil
}

// Execute executes a Redis command with context support
// The statement should be a Redis command like "GET key" or "SET key value"
func (r *RedisConnector) Execute(ctx context.Context, statement string, params map[string]any) ([]map[string]any, error) {
	// Split statement into command and args
	parts := strings.Fields(statement)
	if len(parts) == 0 {
		return nil, fmt.Errorf("empty redis command")
	}

	command := strings.ToUpper(parts[0])
	args := parts[1:]

	// Substitute params in args
	for i, arg := range args {
		for key, value := range params {
			placeholder := fmt.Sprintf("{{ inputs.%s }}", key)
			if strings.Contains(arg, placeholder) {
				args[i] = strings.ReplaceAll(arg, placeholder, fmt.Sprintf("%v", value))
			}
		}
	}

	// Convert args to []interface{}
	cmdArgs := make([]any, len(args))
	for i, arg := range args {
		cmdArgs[i] = arg
	}

	// Execute command with provided context
	cmd := r.client.Do(ctx, append([]any{command}, cmdArgs...)...)
	if cmd.Err() != nil {
		return nil, fmt.Errorf("redis command failed: %w", cmd.Err())
	}

	// Format result as map
	result := make(map[string]any)
	val, err := cmd.Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get result: %w", err)
	}

	// Handle different result types
	switch v := val.(type) {
	case string:
		result["value"] = v
	case []any:
		result["values"] = v
	case map[any]any:
		// Convert to map[string]interface{}
		strMap := make(map[string]any)
		for k, v := range v {
			strMap[fmt.Sprintf("%v", k)] = v
		}
		result["value"] = strMap
	default:
		result["value"] = v
	}

	return []map[string]any{result}, nil
}

// Close closes the Redis connection
func (r *RedisConnector) Close() error {
	if r.client != nil {
		log := logger.New("connector:redis")
		log.Debugf("Closing Redis connection")
		err := r.client.Close()
		if err != nil {
			log.Errorf("Error closing Redis connection: %v", err)
		} else {
			log.Debugf("Redis connection closed")
		}
		return err
	}
	return nil
}
