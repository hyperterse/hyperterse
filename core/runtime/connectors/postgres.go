package connectors

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"strings"

	"github.com/hyperterse/hyperterse/core/logger"
	_ "github.com/lib/pq"
)

// PostgresConnector implements the Connector interface for PostgreSQL
type PostgresConnector struct {
	db *sql.DB
}

// NewPostgresConnector creates a new PostgreSQL connector
func NewPostgresConnector(connectionString string, options map[string]string) (*PostgresConnector, error) {
	// Append all options to connection string if provided
	if options != nil && len(options) > 0 {
		// Check if connection string is URL format (starts with postgres:// or postgresql://)
		if strings.HasPrefix(connectionString, "postgres://") || strings.HasPrefix(connectionString, "postgresql://") {
			// Parse the URL format connection string
			parsedURL, err := url.Parse(connectionString)
			if err != nil {
				return nil, fmt.Errorf("failed to parse postgres connection string: %w", err)
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
		} else {
			// Handle key-value format connection string (e.g., "host=localhost port=5432 ...")
			// Append all options as key-value pairs
			var parts []string
			for key, value := range options {
				parts = append(parts, fmt.Sprintf("%s=%s", key, value))
			}
			if len(parts) > 0 {
				// Append to existing connection string
				if !strings.HasSuffix(connectionString, " ") {
					connectionString += " "
				}
				connectionString += strings.Join(parts, " ")
			}
		}
	}

	log := logger.New("connector:postgres")
	log.Debugf("Opening PostgreSQL connection pool")

	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		log.Errorf("Failed to open PostgreSQL connection: %v", err)
		return nil, fmt.Errorf("failed to open postgres connection: %w", err)
	}

	// Test the connection
	log.Debugf("Testing connection with ping")
	if err := db.Ping(); err != nil {
		db.Close()
		log.Errorf("Failed to ping PostgreSQL database: %v", err)
		return nil, fmt.Errorf("failed to ping postgres database: %w", err)
	}

	log.Debugf("PostgreSQL connection pool opened successfully")
	return &PostgresConnector{db: db}, nil
}

// Execute executes a SQL statement against PostgreSQL with context support
func (p *PostgresConnector) Execute(ctx context.Context, statement string, params map[string]any) ([]map[string]any, error) {
	// Convert params map to ordered slice for parameterized queries
	// For now, we'll use direct substitution since the statement already has {{ inputs.x }} format
	// In production, this should be converted to parameterized queries

	rows, err := p.db.QueryContext(ctx, statement)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
	}

	// Build result slice
	var results []map[string]any
	for rows.Next() {
		// Create slice to hold values
		values := make([]any, len(columns))
		valuePtrs := make([]any, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		// Scan row into values
		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		// Create map for this row
		rowMap := make(map[string]any)
		for i, col := range columns {
			val := values[i]
			// Convert []byte to string for better JSON serialization
			if b, ok := val.([]byte); ok {
				rowMap[col] = string(b)
			} else {
				rowMap[col] = val
			}
		}

		results = append(results, rowMap)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return results, nil
}

// Close closes the database connection
func (p *PostgresConnector) Close() error {
	if p.db != nil {
		log := logger.New("connector:postgres")
		log.Debugf("Closing PostgreSQL connection pool")
		err := p.db.Close()
		if err != nil {
			log.Errorf("Error closing PostgreSQL connection: %v", err)
		} else {
			log.Debugf("PostgreSQL connection pool closed")
		}
		return err
	}
	return nil
}
