package main

import (
	"context"
	"flag"
	"fmt"
	"math/rand"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type order struct {
	ID        string    `bson:"id"`
	Status    string    `bson:"status"`
	Total     float64   `bson:"total"`
	CreatedAt time.Time `bson:"createdAt"`
}

func main() {
	var (
		uri        string
		database   string
		collection string
		count      int
	)
	flag.StringVar(&uri, "uri", "mongodb://localhost:27017", "MongoDB connection string")
	flag.StringVar(&database, "db", "my_mongo", "Database name")
	flag.StringVar(&collection, "collection", "orders", "Collection name")
	flag.IntVar(&count, "count", 25, "Number of orders to generate")
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		panic(fmt.Errorf("connect failed: %w", err))
	}
	defer func() {
		_ = client.Disconnect(context.Background())
	}()

	coll := client.Database(database).Collection(collection)

	statuses := []string{"pending", "paid", "shipped", "cancelled"}
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	docs := make([]any, 0, count)
	for i := 0; i < count; i++ {
		id := fmt.Sprintf("order_%d_%06d", time.Now().Unix(), rng.Intn(1_000_000))
		o := order{
			ID:        id,
			Status:    statuses[rng.Intn(len(statuses))],
			Total:     float64(10+rng.Intn(4900)) / 100.0,
			CreatedAt: time.Now().Add(-time.Duration(rng.Intn(720)) * time.Hour),
		}
		docs = append(docs, o)
	}

	res, err := coll.InsertMany(ctx, docs)
	if err != nil {
		panic(fmt.Errorf("insert failed: %w", err))
	}

	fmt.Printf("inserted %d orders into %s.%s\n", len(res.InsertedIDs), database, collection)
	fmt.Printf("example order id for query: %s\n", docs[0].(order).ID)
}
