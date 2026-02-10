package executor

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/dgraph-io/ristretto"
)

const (
	// Keep up to ~128 MiB of cached query results in-memory.
	defaultRistrettoMaxCost = 128 << 20
	// Rule of thumb from Ristretto: ~10x expected live keys.
	defaultRistrettoNumCounters = 1_000_000
	defaultRistrettoBufferItems = 64
)

type queryCache struct {
	store *ristretto.Cache
}

func newQueryCache() *queryCache {
	// Ristretto requires sizing knobs up front.
	// These defaults are tuned for query-result caching where values are
	// variable-sized row sets and we want good hit ratio without unbounded RAM.
	store, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: defaultRistrettoNumCounters,
		MaxCost:     defaultRistrettoMaxCost,
		BufferItems: defaultRistrettoBufferItems,
	})
	if err != nil {
		// Invalid config should never happen with static values.
		panic(err)
	}

	return &queryCache{
		store: store,
	}
}

func (c *queryCache) Get(key string) ([]map[string]any, bool) {
	value, ok := c.store.Get(key)
	if !ok {
		return nil, false
	}

	rows, ok := value.([]map[string]any)
	if !ok {
		return nil, false
	}

	return cloneRows(rows), true
}

func (c *queryCache) Set(key string, rows []map[string]any, ttl time.Duration) {
	if ttl <= 0 {
		return
	}

	cost := estimateRowsCost(rows)

	clonedRows := cloneRows(rows)
	accepted := c.store.SetWithTTL(key, clonedRows, cost, ttl)
	if accepted {
		// Ristretto sets are asynchronous. Wait ensures the value can be read
		// immediately by the next query execution.
		c.store.Wait()
	}
}

func buildCacheKey(queryName, finalStatement string) string {
	hash := sha256.Sum256([]byte(queryName + ":" + finalStatement))
	return hex.EncodeToString(hash[:])
}

func cloneRows(rows []map[string]any) []map[string]any {
	if rows == nil {
		return nil
	}

	out := make([]map[string]any, len(rows))
	for i, row := range rows {
		if row == nil {
			continue
		}
		copyRow := make(map[string]any, len(row))
		for key, value := range row {
			copyRow[key] = value
		}
		out[i] = copyRow
	}
	return out
}

func estimateRowsCost(rows []map[string]any) int64 {
	if len(rows) == 0 {
		return 1
	}

	var total int64
	for _, row := range rows {
		if row == nil {
			continue
		}
		// Map entry overhead plus key/value estimation.
		total += int64(len(row) * 16)
		for key, value := range row {
			total += int64(len(key))
			total += estimateValueCost(value)
		}
	}

	if total <= 0 {
		return 1
	}
	return total
}

func estimateValueCost(v any) int64 {
	switch val := v.(type) {
	case nil:
		return 0
	case string:
		return int64(len(val))
	case []byte:
		return int64(len(val))
	case bool:
		return 1
	case int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64:
		return 8
	case float32:
		return 4
	case float64:
		return 8
	case time.Time:
		return 16
	case map[string]any:
		var size int64
		for key, nested := range val {
			size += int64(len(key)) + estimateValueCost(nested)
		}
		return size
	case []any:
		var size int64
		for _, nested := range val {
			size += estimateValueCost(nested)
		}
		return size
	default:
		// Fallback for uncommon/custom types.
		return int64(len(fmt.Sprintf("%v", val)))
	}
}
