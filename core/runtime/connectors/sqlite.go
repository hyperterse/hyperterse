package connectors

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/hyperterse/hyperterse/core/logger"
	"github.com/hyperterse/hyperterse/core/observability"
	protoconnectors "github.com/hyperterse/hyperterse/core/proto/connectors"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	_ "modernc.org/sqlite"
)

// SQLiteConnector implements the Connector interface for SQLite/libSQL.
type SQLiteConnector struct {
	db           *sql.DB
	remoteClient *sqliteRemoteClient
}

// NewSQLiteConnector creates a new SQLite connector.
func NewSQLiteConnector(def *protoconnectors.ConnectorDef) (*SQLiteConnector, error) {
	connectionString := def.GetConnectionString()
	options := def.GetOptions()

	if def.GetConfig().GetJsonStatements() {
		return nil, fmt.Errorf("json_statements is not supported for sqlite")
	}

	var err error
	connectionString, err = appendSQLiteOptionsToConnectionString(connectionString, options)
	if err != nil {
		return nil, fmt.Errorf("failed to append sqlite options to connection string: %w", err)
	}

	connectionStringLower := strings.ToLower(connectionString)
	if strings.HasPrefix(connectionStringLower, "ws://") || strings.HasPrefix(connectionStringLower, "wss://") {
		return nil, fmt.Errorf("ws:// and wss:// connection strings are not supported for sqlite yet")
	}

	if strings.HasPrefix(connectionStringLower, "libsql://") ||
		strings.HasPrefix(connectionStringLower, "https://") ||
		strings.HasPrefix(connectionStringLower, "http://") {
		remoteClient, err := newSQLiteRemoteClient(connectionString)
		if err != nil {
			return nil, fmt.Errorf("failed to configure sqlite remote client: %w", err)
		}
		return &SQLiteConnector{remoteClient: remoteClient}, nil
	}

	log := logger.New("connector:sqlite")
	log.Debugf("Opening SQLite database")

	db, err := sql.Open("sqlite", connectionString)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite connection: %w", err)
	}

	log.Debugf("Testing connection with ping")
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to ping sqlite database: %w", err)
	}

	log.Debugf("SQLite connection opened successfully")
	return &SQLiteConnector{db: db}, nil
}

// Execute executes a SQL statement against SQLite/local or libSQL remote.
func (s *SQLiteConnector) Execute(ctx context.Context, statement string, params map[string]any) ([]map[string]any, error) {
	_ = params // Existing connectors currently execute fully-rendered statements.

	start := time.Now()
	tracer := otel.Tracer("hyperterse/runtime/connectors/sqlite")
	ctx, span := tracer.Start(ctx, "connector.sqlite.execute")
	defer span.End()
	span.SetAttributes(attribute.String(observability.AttrConnectorType, "sqlite"))

	if s.remoteClient != nil {
		results, err := s.remoteClient.Execute(ctx, statement)
		if err != nil {
			span.SetStatus(codes.Error, "query_failed")
			observability.RecordConnectorOperation(ctx, "", "sqlite", "execute", false, float64(time.Since(start).Milliseconds()))
			return nil, fmt.Errorf("failed to execute sqlite remote query: %w", err)
		}
		observability.RecordConnectorOperation(ctx, "", "sqlite", "execute", true, float64(time.Since(start).Milliseconds()))
		return results, nil
	}

	if s.db == nil {
		span.SetStatus(codes.Error, "missing_connection")
		observability.RecordConnectorOperation(ctx, "", "sqlite", "execute", false, float64(time.Since(start).Milliseconds()))
		return nil, fmt.Errorf("sqlite connector is not initialized")
	}

	rows, err := s.db.QueryContext(ctx, statement)
	if err != nil {
		span.SetStatus(codes.Error, "query_failed")
		observability.RecordConnectorOperation(ctx, "", "sqlite", "execute", false, float64(time.Since(start).Milliseconds()))
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		span.SetStatus(codes.Error, "columns_failed")
		observability.RecordConnectorOperation(ctx, "", "sqlite", "execute", false, float64(time.Since(start).Milliseconds()))
		return nil, fmt.Errorf("failed to get columns: %w", err)
	}

	var results []map[string]any
	for rows.Next() {
		values := make([]any, len(columns))
		valuePtrs := make([]any, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			span.SetStatus(codes.Error, "scan_failed")
			observability.RecordConnectorOperation(ctx, "", "sqlite", "execute", false, float64(time.Since(start).Milliseconds()))
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		rowMap := make(map[string]any)
		for i, col := range columns {
			val := values[i]
			if b, ok := val.([]byte); ok {
				rowMap[col] = string(b)
			} else {
				rowMap[col] = val
			}
		}
		results = append(results, rowMap)
	}

	if err := rows.Err(); err != nil {
		span.SetStatus(codes.Error, "rows_iteration_failed")
		observability.RecordConnectorOperation(ctx, "", "sqlite", "execute", false, float64(time.Since(start).Milliseconds()))
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	observability.RecordConnectorOperation(ctx, "", "sqlite", "execute", true, float64(time.Since(start).Milliseconds()))
	return results, nil
}

// Close closes the SQLite connection.
func (s *SQLiteConnector) Close() error {
	if s.db != nil {
		log := logger.New("connector:sqlite")
		log.Debugf("Closing SQLite connection")
		err := s.db.Close()
		if err == nil {
			log.Debugf("SQLite connection closed")
		}
		return err
	}
	return nil
}

func appendSQLiteOptionsToConnectionString(connectionString string, options map[string]string) (string, error) {
	if len(options) == 0 {
		return connectionString, nil
	}

	parsedURL, err := url.Parse(connectionString)
	if err == nil && (parsedURL.Scheme != "" || strings.HasPrefix(strings.ToLower(connectionString), "file:")) {
		query := parsedURL.Query()
		for key, value := range options {
			query.Set(key, value)
		}
		parsedURL.RawQuery = query.Encode()
		return parsedURL.String(), nil
	}

	query := url.Values{}
	for key, value := range options {
		query.Set(key, value)
	}

	encoded := query.Encode()
	if encoded == "" {
		return connectionString, nil
	}

	separator := "?"
	if strings.Contains(connectionString, "?") {
		separator = "&"
	}
	return connectionString + separator + encoded, nil
}

type sqliteRemoteClient struct {
	pipelineURL string
	authToken   string
	httpClient  *http.Client
}

func newSQLiteRemoteClient(connectionString string) (*sqliteRemoteClient, error) {
	parsedURL, err := url.Parse(connectionString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse sqlite remote connection string: %w", err)
	}

	query := parsedURL.Query()
	authToken, err := extractSQLiteAuthToken(&query)
	if err != nil {
		return nil, err
	}

	tlsEnabled, err := extractSQLiteTLSEnabled(&query, parsedURL.Scheme)
	if err != nil {
		return nil, err
	}

	if parsedURL.Scheme == "libsql" {
		if tlsEnabled {
			parsedURL.Scheme = "https"
		} else {
			if parsedURL.Port() == "" {
				return nil, fmt.Errorf("libsql:// URL with ?tls=0 must specify an explicit port")
			}
			parsedURL.Scheme = "http"
		}
	}

	if (parsedURL.Scheme == "https") && !tlsEnabled {
		return nil, fmt.Errorf("https:// URL cannot opt out of TLS using ?tls=0")
	}
	if (parsedURL.Scheme == "http") && tlsEnabled {
		return nil, fmt.Errorf("http:// URL cannot opt in to TLS using ?tls=1")
	}

	if parsedURL.Scheme != "https" && parsedURL.Scheme != "http" {
		return nil, fmt.Errorf("unsupported sqlite remote URL scheme: %s", parsedURL.Scheme)
	}

	parsedURL.RawQuery = query.Encode()

	pipelineURL := *parsedURL
	basePath := strings.TrimSuffix(pipelineURL.Path, "/")
	if basePath == "" {
		pipelineURL.Path = "/v2/pipeline"
	} else {
		pipelineURL.Path = basePath + "/v2/pipeline"
	}

	return &sqliteRemoteClient{
		pipelineURL: pipelineURL.String(),
		authToken:   authToken,
		httpClient:  http.DefaultClient,
	}, nil
}

func extractSQLiteAuthToken(query *url.Values) (string, error) {
	authTokenSnake := query.Get("auth_token")
	authTokenCamel := query.Get("authToken")
	jwt := query.Get("jwt")
	query.Del("auth_token")
	query.Del("authToken")
	query.Del("jwt")

	nonEmptyCount := 0
	for _, val := range []string{authTokenSnake, authTokenCamel, jwt} {
		if val != "" {
			nonEmptyCount++
		}
	}
	if nonEmptyCount > 1 {
		return "", fmt.Errorf("please use at most one of the following query parameters: 'auth_token', 'authToken', 'jwt'")
	}

	if authTokenSnake != "" {
		return authTokenSnake, nil
	}
	if authTokenCamel != "" {
		return authTokenCamel, nil
	}
	return jwt, nil
}

func extractSQLiteTLSEnabled(query *url.Values, scheme string) (bool, error) {
	tls := query.Get("tls")
	query.Del("tls")

	switch tls {
	case "":
		return scheme != "http" && scheme != "ws", nil
	case "0":
		return false, nil
	case "1":
		return true, nil
	default:
		return true, fmt.Errorf("unknown value of tls query parameter. Valid values are 0 and 1")
	}
}

func (c *sqliteRemoteClient) Execute(ctx context.Context, statement string) ([]map[string]any, error) {
	stmt := statement
	payload := sqlitePipelineRequest{
		Requests: []sqliteStreamRequest{
			{
				Type: "execute",
				Stmt: &sqliteStmt{
					SQL:      &stmt,
					WantRows: true,
				},
			},
		},
	}

	reqBody, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to encode sqlite remote request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.pipelineURL, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to build sqlite remote request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.authToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.authToken)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sqlite remote request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		var responseErr struct {
			Error   string `json:"error"`
			Message string `json:"message"`
		}
		if err := json.Unmarshal(body, &responseErr); err == nil {
			if responseErr.Error != "" {
				return nil, fmt.Errorf("sqlite remote request failed with status %d: %s", resp.StatusCode, responseErr.Error)
			}
			if responseErr.Message != "" {
				return nil, fmt.Errorf("sqlite remote request failed with status %d: %s", resp.StatusCode, responseErr.Message)
			}
		}
		return nil, fmt.Errorf("sqlite remote request failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var pipelineResponse sqlitePipelineResponse
	if err := json.NewDecoder(resp.Body).Decode(&pipelineResponse); err != nil {
		return nil, fmt.Errorf("failed to decode sqlite remote response: %w", err)
	}
	if len(pipelineResponse.Results) == 0 {
		return nil, fmt.Errorf("sqlite remote response did not include results")
	}

	result := pipelineResponse.Results[0]
	if result.Error != nil {
		if result.Error.Code != nil {
			return nil, fmt.Errorf("sqlite remote error code %s: %s", *result.Error.Code, result.Error.Message)
		}
		return nil, fmt.Errorf("sqlite remote error: %s", result.Error.Message)
	}
	if result.Response == nil {
		return nil, fmt.Errorf("sqlite remote response did not include result payload")
	}
	if result.Response.Type != "execute" {
		return nil, fmt.Errorf("unexpected sqlite remote response type: %s", result.Response.Type)
	}

	var stmtResult sqliteStmtResult
	if err := json.Unmarshal(result.Response.Result, &stmtResult); err != nil {
		return nil, fmt.Errorf("failed to parse sqlite execute result: %w", err)
	}

	return sqliteRowsFromStmtResult(stmtResult), nil
}

func sqliteRowsFromStmtResult(result sqliteStmtResult) []map[string]any {
	rows := make([]map[string]any, 0, len(result.Rows))
	for _, row := range result.Rows {
		rowMap := make(map[string]any, len(result.Cols))
		for idx, col := range result.Cols {
			columnName := fmt.Sprintf("column_%d", idx)
			if col.Name != nil && *col.Name != "" {
				columnName = *col.Name
			}

			if idx >= len(row) {
				rowMap[columnName] = nil
				continue
			}

			rowMap[columnName] = row[idx].toValue(col.Type)
		}
		rows = append(rows, rowMap)
	}
	return rows
}

type sqlitePipelineRequest struct {
	Baton    string                `json:"baton,omitempty"`
	Requests []sqliteStreamRequest `json:"requests"`
}

type sqliteStreamRequest struct {
	Type string      `json:"type"`
	Stmt *sqliteStmt `json:"stmt,omitempty"`
}

type sqliteStmt struct {
	SQL      *string `json:"sql,omitempty"`
	WantRows bool    `json:"want_rows"`
}

type sqlitePipelineResponse struct {
	Baton   string               `json:"baton,omitempty"`
	BaseURL string               `json:"base_url,omitempty"`
	Results []sqliteStreamResult `json:"results"`
}

type sqliteStreamResult struct {
	Response *sqliteStreamResponse `json:"response,omitempty"`
	Error    *sqliteRemoteError    `json:"error,omitempty"`
}

type sqliteStreamResponse struct {
	Type   string          `json:"type"`
	Result json.RawMessage `json:"result,omitempty"`
}

type sqliteRemoteError struct {
	Message string  `json:"message"`
	Code    *string `json:"code,omitempty"`
}

type sqliteStmtResult struct {
	Cols             []sqliteColumn  `json:"cols"`
	Rows             [][]sqliteValue `json:"rows"`
	AffectedRowCount int32           `json:"affected_row_count"`
	LastInsertRowID  *string         `json:"last_insert_rowid"`
}

type sqliteColumn struct {
	Name *string `json:"name"`
	Type *string `json:"decltype"`
}

type sqliteValue struct {
	Type   string  `json:"type"`
	Value  any     `json:"value,omitempty"`
	Base64 *string `json:"base64,omitempty"`
}

func (v sqliteValue) toValue(columnType *string) any {
	switch v.Type {
	case "null":
		return nil
	case "integer":
		switch val := v.Value.(type) {
		case string:
			parsed, err := strconv.ParseInt(val, 10, 64)
			if err != nil {
				return nil
			}
			return parsed
		case float64:
			return int64(val)
		default:
			return nil
		}
	case "float":
		switch val := v.Value.(type) {
		case float64:
			return val
		case string:
			parsed, err := strconv.ParseFloat(val, 64)
			if err != nil {
				return nil
			}
			return parsed
		default:
			return nil
		}
	case "blob":
		if v.Base64 == nil {
			return nil
		}
		data, err := base64.StdEncoding.WithPadding(base64.NoPadding).DecodeString(*v.Base64)
		if err == nil {
			return data
		}
		data, err = base64.StdEncoding.DecodeString(*v.Base64)
		if err != nil {
			return nil
		}
		return data
	case "text":
		text, ok := v.Value.(string)
		if !ok {
			return v.Value
		}
		if columnType == nil {
			return text
		}

		columnTypeLower := strings.ToLower(*columnType)
		if columnTypeLower == "timestamp" || columnTypeLower == "datetime" {
			for _, format := range []string{
				"2006-01-02 15:04:05.999999999-07:00",
				"2006-01-02T15:04:05.999999999-07:00",
				"2006-01-02 15:04:05.999999999",
				"2006-01-02T15:04:05.999999999",
				"2006-01-02 15:04:05",
				"2006-01-02T15:04:05",
				"2006-01-02 15:04",
				"2006-01-02T15:04",
				"2006-01-02",
			} {
				parsedTime, err := time.ParseInLocation(format, text, time.UTC)
				if err == nil {
					return parsedTime
				}
			}
		}
		return text
	default:
		return v.Value
	}
}
