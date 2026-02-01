package connectors

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/hyperterse/hyperterse/core/logger"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
)

// MongoDBConnector implements the Connector interface for MongoDB
type MongoDBConnector struct {
	client *mongo.Client
}

// NewMongoDBConnector creates a new MongoDB connector
func NewMongoDBConnector(connectionString string, optionsMap map[string]string) (*MongoDBConnector, error) {
	log := logger.New("connector:mongodb")
	log.Debugf("Opening MongoDB connection")

	opts := options.Client().ApplyURI(connectionString)

	if optionsMap != nil {
		if v, ok := optionsMap["maxPoolSize"]; ok {
			if n, err := strconv.ParseUint(v, 10, 64); err == nil {
				opts.SetMaxPoolSize(n)
			}
		}
		if v, ok := optionsMap["minPoolSize"]; ok {
			if n, err := strconv.ParseUint(v, 10, 64); err == nil {
				opts.SetMinPoolSize(n)
			}
		}
		if v, ok := optionsMap["connectTimeoutMS"]; ok {
			if n, err := strconv.ParseInt(v, 10, 64); err == nil {
				opts.SetConnectTimeout(time.Duration(n) * time.Millisecond)
			}
		}
		if v, ok := optionsMap["serverSelectionTimeoutMS"]; ok {
			if n, err := strconv.ParseInt(v, 10, 64); err == nil {
				opts.SetServerSelectionTimeout(time.Duration(n) * time.Millisecond)
			}
		}
	}

	client, err := mongo.Connect(opts)
	if err != nil {
		log.Errorf("Failed to connect to MongoDB: %v", err)
		return nil, fmt.Errorf("failed to connect to mongodb: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	log.Debugf("Testing connection with ping")
	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		_ = client.Disconnect(ctx)
		log.Errorf("Failed to ping MongoDB: %v", err)
		return nil, fmt.Errorf("failed to ping mongodb: %w", err)
	}

	log.Debugf("MongoDB connection opened successfully")
	return &MongoDBConnector{client: client}, nil
}

// mongoStatement represents the JSON structure for a MongoDB operation
type mongoStatement struct {
	Database    string         `json:"database"`
	Collection  string         `json:"collection"`
	Operation   string         `json:"operation"`
	Filter      map[string]any `json:"filter"`
	Document    map[string]any `json:"document"`
	Documents   []map[string]any `json:"documents"`
	Update      map[string]any `json:"update"`
	Pipeline    []map[string]any `json:"pipeline"`
	Options     map[string]any `json:"options"`
}

// toBSON converts map[string]any to bson.M for driver calls (handles nested maps and slices)
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

func toBSONSlice(docs []map[string]any) []any {
	if docs == nil {
		return nil
	}
	out := make([]any, len(docs))
	for i, m := range docs {
		out[i] = toBSON(m)
	}
	return out
}

// Execute executes a MongoDB operation. The statement must be a JSON object with database, collection, operation, and operation-specific fields.
func (m *MongoDBConnector) Execute(ctx context.Context, statement string, params map[string]any) ([]map[string]any, error) {
	var stmt mongoStatement
	if err := json.Unmarshal([]byte(statement), &stmt); err != nil {
		return nil, fmt.Errorf("mongodb statement must be valid JSON: %w", err)
	}

	if stmt.Database == "" || stmt.Collection == "" {
		return nil, fmt.Errorf("mongodb statement must include database and collection")
	}
	if stmt.Operation == "" {
		return nil, fmt.Errorf("mongodb statement must include operation (find, findOne, insertOne, insertMany, updateOne, updateMany, deleteOne, deleteMany, aggregate, countDocuments)")
	}

	coll := m.client.Database(stmt.Database).Collection(stmt.Collection)
	filter := toBSON(stmt.Filter)

	switch stmt.Operation {
	case "find":
		return m.executeFind(ctx, coll, filter, stmt.Options)
	case "findOne":
		return m.executeFindOne(ctx, coll, filter, stmt.Options)
	case "insertOne":
		return m.executeInsertOne(ctx, coll, stmt.Document)
	case "insertMany":
		return m.executeInsertMany(ctx, coll, stmt.Documents)
	case "updateOne":
		return m.executeUpdateOne(ctx, coll, filter, stmt.Update, stmt.Options)
	case "updateMany":
		return m.executeUpdateMany(ctx, coll, filter, stmt.Update, stmt.Options)
	case "deleteOne":
		return m.executeDeleteOne(ctx, coll, filter)
	case "deleteMany":
		return m.executeDeleteMany(ctx, coll, filter)
	case "aggregate":
		return m.executeAggregate(ctx, coll, stmt.Pipeline)
	case "countDocuments":
		return m.executeCountDocuments(ctx, coll, filter, stmt.Options)
	default:
		return nil, fmt.Errorf("unsupported mongodb operation: %s", stmt.Operation)
	}
}

func (m *MongoDBConnector) executeFind(ctx context.Context, coll *mongo.Collection, filter bson.M, optsMap map[string]any) ([]map[string]any, error) {
	opts := options.Find()
	if optsMap != nil {
		if v, ok := optsMap["limit"]; ok {
			if n, ok := toInt64(v); ok {
				opts.SetLimit(n)
			}
		}
		if v, ok := optsMap["sort"]; ok {
			if sortMap, ok := v.(map[string]any); ok {
				opts.SetSort(toBSON(sortMap))
			}
		}
		if v, ok := optsMap["projection"]; ok {
			if projMap, ok := v.(map[string]any); ok {
				opts.SetProjection(toBSON(projMap))
			}
		}
		if v, ok := optsMap["skip"]; ok {
			if n, ok := toInt64(v); ok {
				opts.SetSkip(n)
			}
		}
	}

	if filter == nil {
		filter = bson.M{}
	}

	cur, err := coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("mongodb find failed: %w", err)
	}
	defer cur.Close(ctx)

	var results []map[string]any
	for cur.Next(ctx) {
		var doc bson.M
		if err := cur.Decode(&doc); err != nil {
			return nil, fmt.Errorf("mongodb decode failed: %w", err)
		}
		results = append(results, bsonMToMap(doc))
	}
	if err := cur.Err(); err != nil {
		return nil, fmt.Errorf("mongodb find cursor error: %w", err)
	}
	return results, nil
}

func (m *MongoDBConnector) executeFindOne(ctx context.Context, coll *mongo.Collection, filter bson.M, optsMap map[string]any) ([]map[string]any, error) {
	opts := options.FindOne()
	if optsMap != nil {
		if v, ok := optsMap["sort"]; ok {
			if sortMap, ok := v.(map[string]any); ok {
				opts.SetSort(toBSON(sortMap))
			}
		}
		if v, ok := optsMap["projection"]; ok {
			if projMap, ok := v.(map[string]any); ok {
				opts.SetProjection(toBSON(projMap))
			}
		}
		if v, ok := optsMap["skip"]; ok {
			if n, ok := toInt64(v); ok {
				opts.SetSkip(n)
			}
		}
	}

	if filter == nil {
		filter = bson.M{}
	}

	var doc bson.M
	err := coll.FindOne(ctx, filter, opts).Decode(&doc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return []map[string]any{}, nil
		}
		return nil, fmt.Errorf("mongodb findOne failed: %w", err)
	}
	return []map[string]any{bsonMToMap(doc)}, nil
}

func (m *MongoDBConnector) executeInsertOne(ctx context.Context, coll *mongo.Collection, doc map[string]any) ([]map[string]any, error) {
	if doc == nil {
		return nil, fmt.Errorf("insertOne requires document")
	}
	res, err := coll.InsertOne(ctx, toBSON(doc))
	if err != nil {
		return nil, fmt.Errorf("mongodb insertOne failed: %w", err)
	}
	return []map[string]any{{"insertedId": res.InsertedID}}, nil
}

func (m *MongoDBConnector) executeInsertMany(ctx context.Context, coll *mongo.Collection, docs []map[string]any) ([]map[string]any, error) {
	if docs == nil || len(docs) == 0 {
		return nil, fmt.Errorf("insertMany requires documents array")
	}
	res, err := coll.InsertMany(ctx, toBSONSlice(docs))
	if err != nil {
		return nil, fmt.Errorf("mongodb insertMany failed: %w", err)
	}
	return []map[string]any{{"insertedIds": res.InsertedIDs}}, nil
}

func (m *MongoDBConnector) executeUpdateOne(ctx context.Context, coll *mongo.Collection, filter, update bson.M, optsMap map[string]any) ([]map[string]any, error) {
	if filter == nil {
		filter = bson.M{}
	}
	if update == nil {
		return nil, fmt.Errorf("updateOne requires update")
	}
	opts := options.UpdateOne()
	if optsMap != nil {
		if v, ok := optsMap["upsert"]; ok {
			if b, ok := v.(bool); ok {
				opts.SetUpsert(b)
			}
		}
	}
	res, err := coll.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return nil, fmt.Errorf("mongodb updateOne failed: %w", err)
	}
	return []map[string]any{
		{"matchedCount": res.MatchedCount, "modifiedCount": res.ModifiedCount, "upsertedCount": res.UpsertedCount, "upsertedId": res.UpsertedID},
	}, nil
}

func (m *MongoDBConnector) executeUpdateMany(ctx context.Context, coll *mongo.Collection, filter, update bson.M, optsMap map[string]any) ([]map[string]any, error) {
	if filter == nil {
		filter = bson.M{}
	}
	if update == nil {
		return nil, fmt.Errorf("updateMany requires update")
	}
	opts := options.UpdateMany()
	if optsMap != nil {
		if v, ok := optsMap["upsert"]; ok {
			if b, ok := v.(bool); ok {
				opts.SetUpsert(b)
			}
		}
	}
	res, err := coll.UpdateMany(ctx, filter, update, opts)
	if err != nil {
		return nil, fmt.Errorf("mongodb updateMany failed: %w", err)
	}
	return []map[string]any{
		{"matchedCount": res.MatchedCount, "modifiedCount": res.ModifiedCount, "upsertedCount": res.UpsertedCount, "upsertedId": res.UpsertedID},
	}, nil
}

func (m *MongoDBConnector) executeDeleteOne(ctx context.Context, coll *mongo.Collection, filter bson.M) ([]map[string]any, error) {
	if filter == nil {
		filter = bson.M{}
	}
	res, err := coll.DeleteOne(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("mongodb deleteOne failed: %w", err)
	}
	return []map[string]any{{"deletedCount": res.DeletedCount}}, nil
}

func (m *MongoDBConnector) executeDeleteMany(ctx context.Context, coll *mongo.Collection, filter bson.M) ([]map[string]any, error) {
	if filter == nil {
		filter = bson.M{}
	}
	res, err := coll.DeleteMany(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("mongodb deleteMany failed: %w", err)
	}
	return []map[string]any{{"deletedCount": res.DeletedCount}}, nil
}

func (m *MongoDBConnector) executeAggregate(ctx context.Context, coll *mongo.Collection, pipeline []map[string]any) ([]map[string]any, error) {
	if pipeline == nil {
		return nil, fmt.Errorf("aggregate requires pipeline array")
	}
	pipe := make([]bson.M, len(pipeline))
	for i, stage := range pipeline {
		pipe[i] = toBSON(stage)
	}

	cur, err := coll.Aggregate(ctx, pipe)
	if err != nil {
		return nil, fmt.Errorf("mongodb aggregate failed: %w", err)
	}
	defer cur.Close(ctx)

	var results []map[string]any
	for cur.Next(ctx) {
		var doc bson.M
		if err := cur.Decode(&doc); err != nil {
			return nil, fmt.Errorf("mongodb aggregate decode failed: %w", err)
		}
		results = append(results, bsonMToMap(doc))
	}
	if err := cur.Err(); err != nil {
		return nil, fmt.Errorf("mongodb aggregate cursor error: %w", err)
	}
	return results, nil
}

func (m *MongoDBConnector) executeCountDocuments(ctx context.Context, coll *mongo.Collection, filter bson.M, optsMap map[string]any) ([]map[string]any, error) {
	if filter == nil {
		filter = bson.M{}
	}
	opts := options.Count()
	if optsMap != nil {
		if v, ok := optsMap["limit"]; ok {
			if n, ok := toInt64(v); ok {
				opts.SetLimit(n)
			}
		}
		if v, ok := optsMap["skip"]; ok {
			if n, ok := toInt64(v); ok {
				opts.SetSkip(n)
			}
		}
	}
	n, err := coll.CountDocuments(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("mongodb countDocuments failed: %w", err)
	}
	return []map[string]any{{"count": n}}, nil
}

func toInt64(v any) (int64, bool) {
	switch val := v.(type) {
	case float64:
		return int64(val), true
	case int:
		return int64(val), true
	case int64:
		return val, true
	case int32:
		return int64(val), true
	default:
		return 0, false
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
	case bson.A:
		arr := make([]any, len(val))
		for i, item := range val {
			arr[i] = bsonValueToAny(item)
		}
		return arr
	default:
		return v
	}
}

// Close closes the MongoDB connection
func (m *MongoDBConnector) Close() error {
	if m.client != nil {
		log := logger.New("connector:mongodb")
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
