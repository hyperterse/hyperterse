package connectors

import (
	"context"
	"fmt"
	"strings"

	"github.com/redis/go-redis/v9"
)

// RedisConnector implements the Connector interface for Redis
type RedisConnector struct {
	client *redis.Client
	ctx    context.Context
}

// NewRedisConnector creates a new Redis connector
func NewRedisConnector(connectionString string) (*RedisConnector, error) {
	// Parse connection string (format: redis://user:password@host:port/db)
	opt, err := redis.ParseURL(connectionString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse redis connection string: %w", err)
	}

	client := redis.NewClient(opt)
	ctx := context.Background()

	// Test the connection
	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to ping redis: %w", err)
	}

	return &RedisConnector{
		client: client,
		ctx:    ctx,
	}, nil
}

// Execute executes a Redis command
// The statement should be a Redis command like "GET key" or "SET key value"
func (r *RedisConnector) Execute(statement string, params map[string]interface{}) ([]map[string]interface{}, error) {
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
	cmdArgs := make([]interface{}, len(args))
	for i, arg := range args {
		cmdArgs[i] = arg
	}

	// Execute command
	cmd := r.client.Do(r.ctx, append([]interface{}{command}, cmdArgs...)...)
	if cmd.Err() != nil {
		return nil, fmt.Errorf("redis command failed: %w", cmd.Err())
	}

	// Format result as map
	result := make(map[string]interface{})
	val, err := cmd.Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get result: %w", err)
	}

	// Handle different result types
	switch v := val.(type) {
	case string:
		result["value"] = v
	case []interface{}:
		result["values"] = v
	case map[interface{}]interface{}:
		// Convert to map[string]interface{}
		strMap := make(map[string]interface{})
		for k, v := range v {
			strMap[fmt.Sprintf("%v", k)] = v
		}
		result["value"] = strMap
	default:
		result["value"] = v
	}

	return []map[string]interface{}{result}, nil
}

// Close closes the Redis connection
func (r *RedisConnector) Close() error {
	if r.client != nil {
		return r.client.Close()
	}
	return nil
}

