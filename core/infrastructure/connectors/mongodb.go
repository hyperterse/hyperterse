package connectors

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/hyperterse/hyperterse/core/domain/interfaces"
	"github.com/hyperterse/hyperterse/core/infrastructure/logging"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	mongoOptions "go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
)

// MongoDBConnector implements the Connector interface for MongoDB
type MongoDBConnector struct {
	client *mongo.Client
}

// NewMongoDBConnector creates a new MongoDB connector
func NewMongoDBConnector(connectionString string, options map[string]string) (interfaces.Connector, error) {
	log := logging.New("connector:mongodb")
	log.Debugf("Opening MongoDB connection")

	// Append all options to connection string if provided
	if len(options) > 0 {
		// Check if connection string is MongoDB URL format (starts with mongodb:// or mongodb+srv://)
		if strings.HasPrefix(connectionString, "mongodb://") || strings.HasPrefix(connectionString, "mongodb+srv://") {
			// Parse the MongoDB connection string
			parsedURL, err := url.Parse(connectionString)
			if err != nil {
				return nil, fmt.Errorf("failed to parse mongodb connection string: %w", err)
			}

			// Get existing query parameters
			query := parsedURL.Query()

			// Append all options directly to query parameters
			for key, value := range options {
				query.Set(key, value)
			}

			// Rebuild connection string with updated query parameters
			parsedURL.RawQuery = query.Encode()
			connectionString = parsedURL.String()
		}
	}

	opts := mongoOptions.Client().ApplyURI(connectionString)
	client, err := mongo.Connect(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to mongodb: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	log.Debugf("Testing connection with ping")
	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		_ = client.Disconnect(ctx)
		return nil, fmt.Errorf("failed to ping mongodb: %w", err)
	}

	log.Debugf("MongoDB connection opened successfully")
	return &MongoDBConnector{client: client}, nil
}

// toBSON converts map[string]any to bson.D for command execution (order-preserving)
func toBSON(m map[string]any) bson.D {
	if m == nil {
		return nil
	}
	var result bson.D
	for k, v := range m {
		result = append(result, bson.E{Key: k, Value: toBSONValue(v)})
	}
	return result
}

func toBSONValue(v any) any {
	switch val := v.(type) {
	case map[string]any:
		if oid, ok := objectIDFromMap(val); ok {
			return oid
		}
		return toBSON(val)
	case []any:
		arr := make(bson.A, len(val))
		for i, item := range val {
			arr[i] = toBSONValue(item)
		}
		return arr
	default:
		return v
	}
}

// Execute executes a MongoDB command directly. The statement must be a JSON object
// representing a MongoDB command (e.g., {"find": "collectionName", "filter": {...}}).
// This approach is infinitely extensible and supports any MongoDB command.
func (m *MongoDBConnector) Execute(ctx context.Context, statement string, params map[string]any) ([]map[string]any, error) {
	// Parse JSON statement into map
	var cmdMap map[string]any
	if err := json.Unmarshal([]byte(statement), &cmdMap); err != nil {
		return nil, fmt.Errorf("mongodb statement must be valid JSON: %w", err)
	}

	// Extract database name if provided, otherwise we'll need it from the command
	var dbName string
	if db, ok := cmdMap["database"].(string); ok {
		dbName = db
		delete(cmdMap, "database")
	}

	// Convert to BSON.D (order-preserving for MongoDB commands)
	command := toBSON(cmdMap)

	// Determine which database to use
	// If database is specified in command, use it; otherwise we need to infer from connection string
	// For now, we'll require database to be specified in the command
	if dbName == "" {
		return nil, fmt.Errorf("mongodb command must include 'database' field")
	}

	db := m.client.Database(dbName)

	// Determine if this command returns a cursor (find, aggregate) or single result
	// Commands that return cursors: find, aggregate
	// Commands that return single results: insert, update, delete, count, etc.
	hasCursor := false
	for key := range cmdMap {
		if key == "find" || key == "aggregate" {
			hasCursor = true
			break
		}
	}

	if hasCursor {
		// Use RunCommandCursor for commands that return multiple documents
		cursor, err := db.RunCommandCursor(ctx, command)
		if err != nil {
			return nil, fmt.Errorf("mongodb command failed: %w", err)
		}
		defer cursor.Close(ctx)

		var results []map[string]any
		for cursor.Next(ctx) {
			var doc bson.M
			if err := cursor.Decode(&doc); err != nil {
				return nil, fmt.Errorf("mongodb decode failed: %w", err)
			}
			results = append(results, bsonMToMap(doc))
		}
		if err := cursor.Err(); err != nil {
			return nil, fmt.Errorf("mongodb cursor error: %w", err)
		}
		return results, nil
	}

	// Use RunCommand for commands that return single results
	var result bson.M
	if err := db.RunCommand(ctx, command).Decode(&result); err != nil {
		return nil, fmt.Errorf("mongodb command failed: %w", err)
	}

	// Check if command was successful
	if ok, _ := result["ok"].(float64); ok != 1 {
		return nil, fmt.Errorf("mongodb command failed: %v", result)
	}

	// Extract the actual result data (remove MongoDB metadata fields)
	cleanResult := make(map[string]any)
	for k, v := range result {
		if k != "ok" && k != "operationTime" && k != "$clusterTime" && k != "$db" {
			cleanResult[k] = bsonValueToAny(v)
		}
	}

	return []map[string]any{cleanResult}, nil
}


func bsonMToMap(doc bson.M) map[string]any {
	if doc == nil {
		return nil
	}
	out := make(map[string]any, len(doc))
	for k, v := range doc {
		out[k] = bsonValueToAny(v)
	}
	return out
}

func bsonValueToAny(v any) any {
	switch val := v.(type) {
	case bson.M:
		return bsonMToMap(val)
	case bson.D:
		return bsonDToMap(val)
	case bson.A:
		arr := make([]any, len(val))
		for i, item := range val {
			arr[i] = bsonValueToAny(item)
		}
		return arr
	case bson.ObjectID:
		return val.Hex()
	case bson.DateTime:
		return val.Time()
	case bson.Decimal128:
		return val.String()
	default:
		return v
	}
}

func bsonDToMap(doc bson.D) map[string]any {
	if doc == nil {
		return nil
	}
	out := make(map[string]any, len(doc))
	for _, elem := range doc {
		out[elem.Key] = bsonValueToAny(elem.Value)
	}
	return out
}

func objectIDFromMap(m map[string]any) (bson.ObjectID, bool) {
	if len(m) != 1 {
		return bson.ObjectID{}, false
	}
	raw, ok := m["$oid"]
	if !ok {
		return bson.ObjectID{}, false
	}
	s, ok := raw.(string)
	if !ok {
		return bson.ObjectID{}, false
	}
	oid, err := bson.ObjectIDFromHex(s)
	if err != nil {
		return bson.ObjectID{}, false
	}
	return oid, true
}

// Close closes the MongoDB connection
func (m *MongoDBConnector) Close() error {
	if m.client != nil {
		log := logging.New("connector:mongodb")
		log.Debugf("Closing MongoDB connection")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err := m.client.Disconnect(ctx)
		if err != nil {
			log.Errorf("Error closing MongoDB connection: %v", err)
		} else {
			log.Debugf("MongoDB connection closed")
		}
		return err
	}
	return nil
}
