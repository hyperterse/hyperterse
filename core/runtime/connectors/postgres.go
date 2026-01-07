package connectors

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

// PostgresConnector implements the Connector interface for PostgreSQL
type PostgresConnector struct {
	db *sql.DB
}

// NewPostgresConnector creates a new PostgreSQL connector
func NewPostgresConnector(connectionString string) (*PostgresConnector, error) {
	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		return nil, fmt.Errorf("failed to open postgres connection: %w", err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping postgres database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)

	return &PostgresConnector{db: db}, nil
}

// Execute executes a SQL statement against PostgreSQL
func (p *PostgresConnector) Execute(statement string, params map[string]interface{}) ([]map[string]interface{}, error) {
	// Convert params map to ordered slice for parameterized queries
	// For now, we'll use direct substitution since the statement already has {{ inputs.x }} format
	// In production, this should be converted to parameterized queries

	rows, err := p.db.Query(statement)
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
	var results []map[string]interface{}
	for rows.Next() {
		// Create slice to hold values
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		// Scan row into values
		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		// Create map for this row
		rowMap := make(map[string]interface{})
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
		return p.db.Close()
	}
	return nil
}

