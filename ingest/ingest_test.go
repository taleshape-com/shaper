package ingest

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/marcboeker/go-duckdb"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockMsg implements the jetstream.Msg interface for testing
type MockMsg struct {
	subject string
	data    []byte
	acked   bool
}

func (m *MockMsg) Subject() string {
	return m.subject
}

func (m *MockMsg) Data() []byte {
	return m.data
}

func (m *MockMsg) Ack() error {
	m.acked = true
	return nil
}

func (m *MockMsg) DoubleAck(ctx context.Context) error {
	m.acked = true
	return nil
}

func (m *MockMsg) Nak() error {
	return nil
}

func (m *MockMsg) Reply() string {
	return ""
}

func (m *MockMsg) NakWithDelay(delay time.Duration) error {
	return nil
}

func (m *MockMsg) Term() error {
	return nil
}

func (m *MockMsg) TermWithReason(reason string) error {
	return nil
}

func (m *MockMsg) InProgress() error {
	return nil
}

func (m *MockMsg) Metadata() (*jetstream.MsgMetadata, error) {
	return &jetstream.MsgMetadata{
		Sequence: jetstream.SequencePair{
			Stream:   1,
			Consumer: 1,
		},
		Timestamp: time.Now(),
	}, nil
}

func (m *MockMsg) Headers() nats.Header {
	return nil
}

func (m *MockMsg) isAcked() bool {
	return m.acked
}

// Helper function to create a mock message
func createMockMsg(subject string, data map[string]any) jetstream.Msg {
	jsonData, _ := json.Marshal(data)
	return &MockMsg{
		subject: subject,
		data:    jsonData,
	}
}

// TestSetup creates a new in-memory DuckDB database for testing
func setupTestDB(t *testing.T) (*duckdb.Connector, *sqlx.DB) {
	// Create an in-memory DuckDB database
	dbConnector, err := duckdb.NewConnector("", nil)
	require.NoError(t, err, "Failed to create DuckDB connector")

	// Open a connection to the database
	sqlDB := sql.OpenDB(dbConnector)
	db := sqlx.NewDb(sqlDB, "duckdb")

	return dbConnector, db
}

func TestDetectSchemaFromBatch(t *testing.T) {
	// Create a batch of messages with different data types
	batch := []jetstream.Msg{
		createMockMsg("test.users", map[string]any{
			"id":        1,
			"name":      "John Doe",
			"is_active": true,
			"created":   "2023-01-15T10:30:45Z",
			"tags":      []string{"tag1", "tag2"},
			"metadata":  map[string]any{"key": "value"},
		}),
		createMockMsg("test.users", map[string]any{
			"id":         2,
			"name":       "Jane Smith",
			"is_active":  false,
			"created":    "2023-02-20T14:15:30Z",
			"score":      95.5,
			"birth_date": "1990-05-15",
		}),
	}

	// Detect schema from batch
	schema, err := detectSchemaFromBatch(batch)
	require.NoError(t, err, "Failed to detect schema from batch")

	// Verify the detected schema
	assert.Equal(t, SQL_TYPE_DOUBLE, schema["id"])
	assert.Equal(t, SQL_TYPE_VARCHAR, schema["name"])
	assert.Equal(t, SQL_TYPE_BOOLEAN, schema["is_active"])
	assert.Equal(t, SQL_TYPE_TIMESTAMP, schema["created"])
	assert.Equal(t, SQL_TYPE_JSON, schema["tags"])
	assert.Equal(t, SQL_TYPE_JSON, schema["metadata"])
	assert.Equal(t, SQL_TYPE_DOUBLE, schema["score"])
	assert.Equal(t, SQL_TYPE_DATE, schema["birth_date"])
}

func TestDetermineColumnType(t *testing.T) {
	testCases := []struct {
		name     string
		samples  []any
		expected string
	}{
		{
			name:     "Boolean values",
			samples:  []any{true, false, true, nil},
			expected: SQL_TYPE_BOOLEAN,
		},
		{
			name:     "Integer values",
			samples:  []any{1, 2, 3, nil},
			expected: SQL_TYPE_DOUBLE,
		},
		{
			name:     "Float values",
			samples:  []any{1.1, 2.2, 3.3, nil},
			expected: SQL_TYPE_DOUBLE,
		},
		{
			name:     "String values",
			samples:  []any{"abc", "def", "ghi", nil},
			expected: SQL_TYPE_VARCHAR,
		},
		{
			name:     "Date values",
			samples:  []any{"2023-01-01", "2023-02-15", nil},
			expected: SQL_TYPE_DATE,
		},
		{
			name:     "Timestamp values",
			samples:  []any{"2023-01-01T12:30:45Z", "2023-02-15T08:15:30Z", nil},
			expected: SQL_TYPE_TIMESTAMP,
		},
		{
			name:     "Array values",
			samples:  []any{[]any{1, 2, 3}, []any{"a", "b", "c"}, nil},
			expected: SQL_TYPE_JSON,
		},
		{
			name:     "Object values",
			samples:  []any{map[string]any{"a": 1}, map[string]any{"b": 2}, nil},
			expected: SQL_TYPE_JSON,
		},
		{
			name:     "Mixed types",
			samples:  []any{"abc", 123, true, nil},
			expected: SQL_TYPE_JSON, // Default to JSON for mixed types
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := determineColumnType(tc.samples)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestTimestampAndDateDetection(t *testing.T) {
	// Test timestamp detection
	timestampTests := map[string]bool{
		"2023-01-01T12:30:45Z":     true,
		"2023-01-01 12:30:45":      true,
		"2023-01-01T12:30:45.123Z": true,
		"2023-01-01":               false,
		"01/02/2023":               false,
		"not a timestamp":          false,
	}

	for input, expected := range timestampTests {
		t.Run("isTimestamp_"+input, func(t *testing.T) {
			result := isTimestamp(input)
			assert.Equal(t, expected, result)
		})
	}

	// Test date detection
	dateTests := map[string]bool{
		"2023-01-01T12:30:45Z": false,
		"2023-01-01 12:30:45":  false,
		"2023-01-01":           true,
		"01/02/2023":           true,
		"not a date":           false,
	}

	for input, expected := range dateTests {
		t.Run("isDate_"+input, func(t *testing.T) {
			result := isDate(input)
			assert.Equal(t, expected, result)
		})
	}
}

func TestProcessBatch(t *testing.T) {
	dbConnector, db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	tableCache := make(map[string]TableCache)
	subjectPrefix := "test."

	// Create a batch of messages for a table
	tableName := "users"
	batch := []jetstream.Msg{
		createMockMsg("test.users", map[string]any{
			"id":        1,
			"name":      "John Doe",
			"is_active": true,
			"created":   "2023-01-15T10:30:45Z",
			"metadata":  map[string]any{"role": "admin"},
		}),
		createMockMsg("test.users", map[string]any{
			"id":        2,
			"name":      "Jane Smith",
			"is_active": false,
			"created":   "2023-02-20T14:15:30Z",
			"metadata":  map[string]any{"role": "user"},
		}),
	}

	// Process the batch
	err := processBatch(ctx, batch, tableCache, dbConnector, db, subjectPrefix)
	require.NoError(t, err, "Failed to process batch")

	// Verify the table was created and data was inserted
	var count int
	err = db.GetContext(ctx, &count, "SELECT COUNT(*) FROM users")
	require.NoError(t, err, "Failed to query user count")
	assert.Equal(t, 2, count, "Expected 2 rows in users table")

	// Verify the table columns
	var columns []ColInfo
	err = db.SelectContext(ctx, &columns, tableColumnsQuery, tableName)
	require.NoError(t, err, "Failed to get table columns")

	// Verify the column types
	columnTypes := make(map[string]string)
	for _, col := range columns {
		columnTypes[col.ColumnName] = col.Type
	}

	assert.Contains(t, columnTypes, "id")
	assert.Contains(t, columnTypes, "name")
	assert.Contains(t, columnTypes, "is_active")
	assert.Contains(t, columnTypes, "created")
	assert.Contains(t, columnTypes, "metadata")

	// Verify the data
	var users []struct {
		ID        int       `db:"id"`
		Name      string    `db:"name"`
		IsActive  bool      `db:"is_active"`
		CreatedAt time.Time `db:"created"`
		Metadata  string    `db:"metadata"`
	}
	err = db.SelectContext(ctx, &users, "SELECT id, name, is_active, created, metadata FROM users ORDER BY id")
	require.NoError(t, err, "Failed to query users")
	assert.Len(t, users, 2, "Expected 2 users")

	// Check if all messages were acknowledged
	for _, msg := range batch {
		mockMsg, ok := msg.(*MockMsg)
		require.True(t, ok, "Failed to cast to MockMsg")
		assert.True(t, mockMsg.isAcked(), "Message was not acknowledged")
	}
}

func TestProcessBatchWithMultipleTables(t *testing.T) {
	dbConnector, db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	tableCache := make(map[string]TableCache)
	subjectPrefix := "test."

	// Create a batch of messages for multiple tables
	batch := []jetstream.Msg{
		createMockMsg("test.users", map[string]any{
			"id":        1,
			"name":      "John Doe",
			"is_active": true,
		}),
		createMockMsg("test.products", map[string]any{
			"id":       101,
			"name":     "Product A",
			"price":    29.99,
			"in_stock": true,
		}),
		createMockMsg("test.users", map[string]any{
			"id":        2,
			"name":      "Jane Smith",
			"is_active": false,
		}),
		createMockMsg("test.products", map[string]any{
			"id":       102,
			"name":     "Product B",
			"price":    49.99,
			"in_stock": false,
		}),
	}

	// Process the batch
	err := processBatch(ctx, batch, tableCache, dbConnector, db, subjectPrefix)
	require.NoError(t, err, "Failed to process batch")

	// Verify users table
	var userCount int
	err = db.GetContext(ctx, &userCount, "SELECT COUNT(*) FROM users")
	require.NoError(t, err, "Failed to query user count")
	assert.Equal(t, 2, userCount, "Expected 2 rows in users table")

	// Verify products table
	var productCount int
	err = db.GetContext(ctx, &productCount, "SELECT COUNT(*) FROM products")
	require.NoError(t, err, "Failed to query product count")
	assert.Equal(t, 2, productCount, "Expected 2 rows in products table")

	// Check if all messages were acknowledged
	for _, msg := range batch {
		mockMsg, ok := msg.(*MockMsg)
		require.True(t, ok, "Failed to cast to MockMsg")
		assert.True(t, mockMsg.isAcked(), "Message was not acknowledged")
	}
}

func TestProcessBatchWithNestedJsonData(t *testing.T) {
	dbConnector, db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	tableCache := make(map[string]TableCache)
	subjectPrefix := "test."

	// Create a batch of messages with nested JSON data
	batch := []jetstream.Msg{
		createMockMsg("test.customers", map[string]any{
			"id":   1,
			"name": "John Doe",
			"address": map[string]any{
				"street":  "123 Main St",
				"city":    "Anytown",
				"country": "USA",
				"zip":     "12345",
			},
			"orders": []any{
				map[string]any{
					"order_id": "A001",
					"amount":   99.99,
				},
				map[string]any{
					"order_id": "A002",
					"amount":   149.99,
				},
			},
		}),
		createMockMsg("test.customers", map[string]any{
			"id":   2,
			"name": "Jane Smith",
			"address": map[string]any{
				"street":  "456 Oak Ave",
				"city":    "Othertown",
				"country": "Canada",
				"zip":     "67890",
			},
			"orders": []any{
				map[string]any{
					"order_id": "B001",
					"amount":   79.99,
				},
			},
		}),
	}

	// Process the batch
	err := processBatch(ctx, batch, tableCache, dbConnector, db, subjectPrefix)
	require.NoError(t, err, "Failed to process batch")

	// Verify the table was created and data was inserted
	var count int
	err = db.GetContext(ctx, &count, "SELECT COUNT(*) FROM customers")
	require.NoError(t, err, "Failed to query customer count")
	assert.Equal(t, 2, count, "Expected 2 rows in customers table")

	// Query and verify the JSON data
	type CustomerRow struct {
		ID      int    `db:"id"`
		Name    string `db:"name"`
		Address string `db:"address"`
		Orders  string `db:"orders"`
	}

	var customers []CustomerRow
	err = db.SelectContext(ctx, &customers, "SELECT id, name, address, orders FROM customers ORDER BY id")
	require.NoError(t, err, "Failed to query customers")
	assert.Len(t, customers, 2, "Expected 2 customers")

	// Parse and verify JSON data for first customer
	var address1 map[string]any
	err = json.Unmarshal([]byte(customers[0].Address), &address1)
	require.NoError(t, err, "Failed to parse address JSON")
	assert.Equal(t, "123 Main St", address1["street"])
	assert.Equal(t, "Anytown", address1["city"])

	var orders1 []map[string]any
	err = json.Unmarshal([]byte(customers[0].Orders), &orders1)
	require.NoError(t, err, "Failed to parse orders JSON")
	assert.Len(t, orders1, 2, "Expected 2 orders for first customer")
	assert.Equal(t, "A001", orders1[0]["order_id"])
}
