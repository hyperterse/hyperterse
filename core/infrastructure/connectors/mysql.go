package connectors

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	"github.com/hyperterse/hyperterse/core/domain/interfaces"
	"github.com/hyperterse/hyperterse/core/infrastructure/logging"
)

// MySQLConnector implements the Connector interface for MySQL
type MySQLConnector struct {
	db *sql.DB
}

// NewMySQLConnector creates a new MySQL connector
func NewMySQLConnector(connectionString string, options map[string]string) (interfaces.Connector, error) {
	// Convert URL format (mysql://user:pass@host:port/db) to DSN format (user:pass@tcp(host:port)/db)
	if strings.HasPrefix(connectionString, "mysql://") {
		parsedURL, err := url.Parse(connectionString)
		if err != nil {
			return nil, fmt.Errorf("failed to parse mysql connection string: %w", err)
		}

		// Extract components
		user := parsedURL.User.Username()
		password, _ := parsedURL.User.Password()
		host := parsedURL.Hostname()
		port := parsedURL.Port()
		if port == "" {
			port = "3306" // Default MySQL port
		}
		database := strings.TrimPrefix(parsedURL.Path, "/")

		// Build DSN format: user:password@tcp(host:port)/database
		var dsn strings.Builder
		if password != "" {
			dsn.WriteString(fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", user, password, host, port, database))
		} else {
			dsn.WriteString(fmt.Sprintf("%s@tcp(%s:%s)/%s", user, host, port, database))
		}

		// Get existing query parameters from URL
		query := parsedURL.Query()

		// Append all options to query parameters
		if options != nil && len(options) > 0 {
			for key, value := range options {
				query.Set(key, value)
			}
		}

		// Append query parameters to DSN if any exist
		if len(query) > 0 {
			dsn.WriteString("?")
			dsn.WriteString(query.Encode())
		}

		connectionString = dsn.String()
	} else if options != nil && len(options) > 0 {
		// Handle DSN format with options - append as query parameters
		var queryParts []string
		for key, value := range options {
			queryParts = append(queryParts, fmt.Sprintf("%s=%s", key, value))
		}
		if len(queryParts) > 0 {
			if strings.Contains(connectionString, "?") {
				connectionString += "&" + strings.Join(queryParts, "&")
			} else {
				connectionString += "?" + strings.Join(queryParts, "&")
			}
		}
	}

	log := logging.New("connector:mysql")
	log.Debugf("Opening MySQL connection pool")

	db, err := sql.Open("mysql", connectionString)
	if err != nil {
		return nil, fmt.Errorf("failed to open mysql connection: %w", err)
	}

	// Test the connection
	log.Debugf("Testing connection with ping")
	if err := db.Ping(); err != nil {
		db.Close()
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
		log := logging.New("connector:mysql")
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
