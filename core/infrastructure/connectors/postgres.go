package connectors

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	
	"github.com/hyperterse/hyperterse/core/domain/interfaces"
	"github.com/hyperterse/hyperterse/core/infrastructure/logging"
)

// PostgresConnector implements the Connector interface for PostgreSQL using pgx/v5
type PostgresConnector struct {
	pool *pgxpool.Pool
}

// NewPostgresConnector creates a new PostgreSQL connector using pgx/v5
func NewPostgresConnector(connectionString string, options map[string]string) (interfaces.Connector, error) {
	// Append all options to connection string if provided
	if options != nil && len(options) > 0 {
		if strings.HasPrefix(connectionString, "postgres://") || strings.HasPrefix(connectionString, "postgresql://") {
			parsedURL, err := url.Parse(connectionString)
			if err != nil {
				return nil, fmt.Errorf("failed to parse postgres connection string: %w", err)
			}

			query := parsedURL.Query()
			for key, value := range options {
				query.Set(key, value)
			}
			parsedURL.RawQuery = query.Encode()
			connectionString = parsedURL.String()
		} else {
			var parts []string
			for key, value := range options {
				parts = append(parts, fmt.Sprintf("%s=%s", key, value))
			}
			if len(parts) > 0 {
				if !strings.HasSuffix(connectionString, " ") {
					connectionString += " "
				}
				connectionString += strings.Join(parts, " ")
			}
		}
	}

	log := logging.New("connector:postgres")
	log.Debugf("Opening PostgreSQL connection pool (pgx/v5)")

	// Parse connection string and create pool config
	config, err := pgxpool.ParseConfig(connectionString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse postgres connection string: %w", err)
	}

	// Create connection pool
	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, fmt.Errorf("failed to create postgres connection pool: %w", err)
	}

	// Test the connection
	log.Debugf("Testing connection with ping")
	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping postgres database: %w", err)
	}

	log.Debugf("PostgreSQL connection pool opened successfully")
	return &PostgresConnector{pool: pool}, nil
}

// Execute executes a SQL statement against PostgreSQL with context support
func (p *PostgresConnector) Execute(ctx context.Context, statement string, params map[string]any) ([]map[string]any, error) {
	rows, err := p.pool.Query(ctx, statement)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	// Get column names
	fieldDescriptions := rows.FieldDescriptions()
	columns := make([]string, len(fieldDescriptions))
	for i, fd := range fieldDescriptions {
		columns[i] = string(fd.Name)
	}

	// Build result slice
	var results []map[string]any
	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
			return nil, fmt.Errorf("failed to get row values: %w", err)
		}

		// Create map for this row
		rowMap := make(map[string]any)
		for i, col := range columns {
			if i < len(values) {
				rowMap[col] = values[i]
			}
		}

		results = append(results, rowMap)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return results, nil
}

// Close closes the database connection pool
func (p *PostgresConnector) Close() error {
	if p.pool != nil {
		log := logging.New("connector:postgres")
		log.Debugf("Closing PostgreSQL connection pool")
		p.pool.Close()
		log.Debugf("PostgreSQL connection pool closed")
	}
	return nil
}
