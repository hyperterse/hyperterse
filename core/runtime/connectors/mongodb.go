package connectors

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/hyperterse/hyperterse/core/logger"
	protoconnectors "github.com/hyperterse/hyperterse/core/proto/connectors"
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
func NewMongoDBConnector(def *protoconnectors.ConnectorDef) (*MongoDBConnector, error) {
	connectionString := def.GetConnectionString()
	options := def.GetOptions()

	if !def.GetConfig().GetJsonStatements() {
		return nil, fmt.Errorf("json_statements must be true for mongodb")
	}

	log := logger.New("connector:mongodb")
	log.Debugf("Opening MongoDB connection")

	// Append all options to connection string if provided
	if len(options) > 0 {
		if strings.HasPrefix(connectionString, "mongodb://") || strings.HasPrefix(connectionString, "mongodb+srv://") {
			parsedURL, err := url.Parse(connectionString)
			if err != nil {
				return nil, fmt.Errorf("failed to parse mongodb connection string: %w", err)
			}
			query := parsedURL.Query()
			for key, value := range options {
				query.Set(key, value)
			}
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

// mongoStatement represents the JSON structure for a MongoDB command.
// Command is kept as json.RawMessage so we can parse it into an ordered bson.D,
// which is required by RunCommand (the command name must be the first key).
type mongoStatement struct {
	Database string          `json:"database"`
	Command  json.RawMessage `json:"command"`
}

// Execute runs a raw MongoDB command via RunCommand.
// The statement must be JSON with "database" and "command" fields.
//
// Example statements:
//
//	{ "database": "mydb", "command": { "find": "orders", "filter": {} } }
//	{ "database": "mydb", "command": { "find": "orders", "filter": { "id": "123" }, "limit": 1, "singleBatch": true } }
//	{ "database": "mydb", "command": { "insert": "orders", "documents": [{ "id": "456", "total": 100 }] } }
//	{ "database": "mydb", "command": { "update": "orders", "updates": [{ "q": { "id": "123" }, "u": { "$set": { "total": 200 } } }] } }
//	{ "database": "mydb", "command": { "delete": "orders", "deletes": [{ "q": { "id": "123" }, "limit": 1 }] } }
//	{ "database": "mydb", "command": { "aggregate": "orders", "pipeline": [{ "$match": { "total": { "$gt": 50 } } }], "cursor": {} } }
//	{ "database": "mydb", "command": { "count": "orders", "query": {} } }
func (m *MongoDBConnector) Execute(ctx context.Context, statement string, params map[string]any) ([]map[string]any, error) {
	var stmt mongoStatement
	if err := json.Unmarshal([]byte(statement), &stmt); err != nil {
		return nil, fmt.Errorf("mongodb statement must be valid JSON: %w", err)
	}

	if stmt.Database == "" {
		return nil, fmt.Errorf("mongodb statement must include database")
	}
	if len(stmt.Command) == 0 {
		return nil, fmt.Errorf("mongodb statement must include command")
	}

	cmd, err := commandToBsonD(stmt.Command)
	if err != nil {
		return nil, fmt.Errorf("invalid mongodb command: %w", err)
	}

	db := m.client.Database(stmt.Database)

	var result bson.M
	if err := db.RunCommand(ctx, cmd).Decode(&result); err != nil {
		return nil, fmt.Errorf("mongodb command failed: %w", err)
	}

	// If the result contains a cursor (find/aggregate), extract documents from firstBatch
	if cursor, ok := result["cursor"]; ok {
		if cursorDoc, ok := cursor.(bson.M); ok {
			if firstBatch, ok := cursorDoc["firstBatch"]; ok {
				if docs, ok := firstBatch.(bson.A); ok {
					results := make([]map[string]any, 0, len(docs))
					for _, doc := range docs {
						if m, ok := doc.(bson.M); ok {
							results = append(results, bsonMToMap(m))
						}
					}
					return results, nil
				}
			}
		}
	}

	// For non-cursor results (insert, update, delete, count, etc.), return the raw result
	return []map[string]any{bsonMToMap(result)}, nil
}

// commandToBsonD parses command JSON into bson.D preserving key order.
// RunCommand requires the command name (e.g. "find") to be the first key.
func commandToBsonD(raw json.RawMessage) (bson.D, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))

	// expect opening {
	t, err := dec.Token()
	if err != nil {
		return nil, err
	}
	if delim, ok := t.(json.Delim); !ok || delim != '{' {
		return nil, fmt.Errorf("command must be a JSON object")
	}

	var d bson.D
	for dec.More() {
		t, err := dec.Token()
		if err != nil {
			return nil, err
		}
		key := t.(string)

		var val any
		if err := dec.Decode(&val); err != nil {
			return nil, err
		}
		d = append(d, bson.E{Key: key, Value: toBSONValue(val)})
	}

	return d, nil
}

// toBSON converts map[string]any to bson.M (handles nested maps and slices)
func toBSON(m map[string]any) bson.M {
	if m == nil {
		return nil
	}
	out := make(bson.M, len(m))
	for k, v := range m {
		out[k] = toBSONValue(v)
	}
	return out
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
		log := logger.New("connector:mongodb")
		log.Debugf("Closing MongoDB connection")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err := m.client.Disconnect(ctx)
		if err == nil {
			log.Debugf("MongoDB connection closed")
		}
		return err
	}
	return nil
}
