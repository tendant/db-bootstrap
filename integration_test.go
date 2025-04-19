package dbstrap

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIntegration runs integration tests against a real PostgreSQL database
// To run these tests, make sure PostgreSQL is running (e.g., via docker-compose)
// and set the DATABASE_URL environment variable
func TestIntegration(t *testing.T) {
	// Skip if not in integration test mode
	if os.Getenv("INTEGRATION_TEST") != "true" {
		t.Skip("Skipping integration test; set INTEGRATION_TEST=true to run")
	}

	// Get database URL from environment
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Fatal("DATABASE_URL must be set for integration tests")
	}

	// Set test password
	os.Setenv("TEST_USER_PASSWORD", "testpass123")
	defer os.Unsetenv("TEST_USER_PASSWORD")

	// Create test config
	yamlData := []byte(`
users:
  - name: dbstrap_test_user
    password_env: TEST_USER_PASSWORD
    can_login: true
    owns_schemas:
      - public
      - test_schema
    roles: []
  - name: dbstrap_readonly_user
    password_env: TEST_USER_PASSWORD
    can_login: true
    owns_schemas: []
    roles: [dbstrap_readonly_role]

databases:
  - name: dbstrap_test_db
    owner: dbstrap_test_user
    encoding: UTF8
    lc_collate: en_US.UTF-8
    lc_ctype: en_US.UTF-8
    template: template0
    extensions:
      - "uuid-ossp"
    grants:
      - user: dbstrap_test_user
        privileges: [CONNECT]
      - user: dbstrap_readonly_user
        privileges: [CONNECT]
    schemas:
      - name: public
        owner: dbstrap_test_user
        grants:
          - user: dbstrap_test_user
            privileges: [USAGE, CREATE]
            table_privileges: [SELECT, INSERT, UPDATE, DELETE]
            sequence_privileges: [USAGE, SELECT, UPDATE]
            function_privileges: [EXECUTE]
            default_privileges: [SELECT, INSERT, UPDATE, DELETE]
          - role: dbstrap_readonly_role
            privileges: [USAGE]
            table_privileges: [SELECT]
            sequence_privileges: [USAGE, SELECT]
            function_privileges: [EXECUTE]
            default_privileges: [SELECT]
      - name: test_schema
        owner: dbstrap_test_user
        grants:
          - user: dbstrap_test_user
            privileges: [USAGE, CREATE]
            table_privileges: [SELECT, INSERT, UPDATE, DELETE]
            sequence_privileges: [USAGE, SELECT, UPDATE]
            function_privileges: [EXECUTE]
            default_privileges: [SELECT, INSERT, UPDATE, DELETE]
          - role: dbstrap_readonly_role
            privileges: [USAGE]
            table_privileges: [SELECT]
            sequence_privileges: [USAGE, SELECT]
            function_privileges: [EXECUTE]
            default_privileges: [SELECT]
`)

	// Run bootstrap
	err := BootstrapDatabase(yamlData)
	require.NoError(t, err)

	// Connect to the test database
	ctx := context.Background()
	
	// First connect to the default database to verify the test database was created
	conn, err := pgx.Connect(ctx, dbURL)
	require.NoError(t, err)
	defer conn.Close(ctx)

	// Verify test database exists
	var dbExists bool
	err = conn.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)", "dbstrap_test_db").Scan(&dbExists)
	require.NoError(t, err)
	assert.True(t, dbExists, "Test database should exist")

	// Verify users exist
	var userExists bool
	err = conn.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM pg_roles WHERE rolname = $1)", "dbstrap_test_user").Scan(&userExists)
	require.NoError(t, err)
	assert.True(t, userExists, "Test user should exist")

	var readonlyUserExists bool
	err = conn.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM pg_roles WHERE rolname = $1)", "dbstrap_readonly_user").Scan(&readonlyUserExists)
	require.NoError(t, err)
	assert.True(t, readonlyUserExists, "Readonly user should exist")

	// Connect to the test database to verify schemas and grants
	testDbConfig, err := pgx.ParseConfig(dbURL)
	require.NoError(t, err)
	testDbConfig.Database = "dbstrap_test_db"
	
	testConn, err := pgx.ConnectConfig(ctx, testDbConfig)
	require.NoError(t, err)
	defer testConn.Close(ctx)

	// Verify test_schema exists
	var schemaExists bool
	err = testConn.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM information_schema.schemata WHERE schema_name = $1)", "test_schema").Scan(&schemaExists)
	require.NoError(t, err)
	assert.True(t, schemaExists, "Test schema should exist")

	// Verify extension was created
	var extensionExists bool
	err = testConn.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM pg_extension WHERE extname = $1)", "uuid-ossp").Scan(&extensionExists)
	require.NoError(t, err)
	assert.True(t, extensionExists, "UUID extension should exist")

	// Create a test table to verify default privileges
	_, err = testConn.Exec(ctx, "CREATE TABLE test_schema.test_table (id serial PRIMARY KEY, name text)")
	require.NoError(t, err)

	// Verify grants by connecting as the readonly user
	readonlyDbConfig, err := pgx.ParseConfig(dbURL)
	require.NoError(t, err)
	readonlyDbConfig.Database = "dbstrap_test_db"
	readonlyDbConfig.User = "dbstrap_readonly_user"
	readonlyDbConfig.Password = "testpass123"
	
	readonlyConn, err := pgx.ConnectConfig(ctx, readonlyDbConfig)
	require.NoError(t, err)
	defer readonlyConn.Close(ctx)

	// Verify readonly user can SELECT but not INSERT
	var count int
	err = readonlyConn.QueryRow(ctx, "SELECT COUNT(*) FROM test_schema.test_table").Scan(&count)
	assert.NoError(t, err, "Readonly user should be able to SELECT")

	_, err = readonlyConn.Exec(ctx, "INSERT INTO test_schema.test_table (name) VALUES ('test')")
	assert.Error(t, err, "Readonly user should not be able to INSERT")

	// Clean up
	testConn.Exec(ctx, "DROP TABLE test_schema.test_table")
	conn.Exec(ctx, "DROP DATABASE dbstrap_test_db")
	conn.Exec(ctx, "DROP ROLE dbstrap_test_user")
	conn.Exec(ctx, "DROP ROLE dbstrap_readonly_user")
	conn.Exec(ctx, "DROP ROLE dbstrap_readonly_role")
}
