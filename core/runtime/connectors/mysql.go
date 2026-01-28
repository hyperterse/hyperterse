package connectors

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"

	"github.com/hyperterse/hyperterse/core/logger"
	_ "github.com/go-sql-driver/mysql"
)

// MySQLConnector implements the Connector interface for MySQL
type MySQLConnector struct {
	db *sql.DB
}

// NewMySQLConnector creates a new MySQL connector
func NewMySQLConnector(connectionString string, options map[string]string) (*MySQLConnector, error) {
	// Append all options to connection string if provided
	if options != nil && len(options) > 0 {
		// Parse the connection string
		parsedURL, err := url.Parse(connectionString)
		if err != nil {
			return nil, fmt.Errorf("failed to parse mysql connection string: %w", err)
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

	log := logger.New("connector:mysql")
	log.Debugf("Opening MySQL connection pool")

	db, err := sql.Open("mysql", connectionString)
	if err != nil {
		log.Errorf("Failed to open MySQL connection: %v", err)
		return nil, fmt.Errorf("failed to open mysql connection: %w", err)
	}

	// Test the connection
	log.Debugf("Testing connection with ping")
	if err := db.Ping(); err != nil {
		db.Close()
		log.Errorf("Failed to ping MySQL database: %v", err)
		return nil, fmt.Errorf("failed to ping mysql database: %w", err)
	}

	log.Debugf("MySQL connection pool opened successfully")
	return &MySQLConnector{db: db}, nil
}

// Execute executes a SQL statement against MySQL with context support
func (m *MySQLConnector) Execute(ctx context.Context, statement string, params map[string]any) ([]map[string]any, error) {
	rows, err := m.db.QueryContext(ctx, statement)
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
func (m *MySQLConnector) Close() error {
	if m.db != nil {
		log := logger.New("connector:mysql")
		log.Debugf("Closing MySQL connection pool")
		err := m.db.Close()
		if err != nil {
			log.Errorf("Error closing MySQL connection: %v", err)
		} else {
			log.Debugf("MySQL connection pool closed")
		}
		return err
	}
	return nil
}
