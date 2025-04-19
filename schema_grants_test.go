package dbstrap

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSchemaGrants tests the schema grants functionality including the new features:
// - Role-based grants
// - Table privileges
// - Sequence privileges
// - Function privileges
// - Default privileges
func TestSchemaGrants(t *testing.T) {
	// Skip if not in integration test mode
	if os.Getenv("INTEGRATION_TEST") != "true" {
		t.Skip("Skipping integration test; set INTEGRATION_TEST=true to run")
	}

	// Get database URL from environment
	dbURL, cleanup := setupTestDatabase(t)
	defer cleanup()

	// Set test password
	os.Setenv("TEST_USER_PASSWORD", "testpass123")
	defer os.Unsetenv("TEST_USER_PASSWORD")

	// Create test config with all the new grant types
	yamlData := []byte(`
users:
  - name: grant_test_user
    password_env: TEST_USER_PASSWORD
    can_login: true
    owns_schemas:
      - grant_test_schema
    roles: []
  - name: grant_readonly_user
    password_env: TEST_USER_PASSWORD
    can_login: true
    owns_schemas: []
    roles: [grant_readonly_role]

databases:
  - name: grant_test_db
    owner: grant_test_user
    encoding: UTF8
    lc_collate: en_US.UTF-8
    lc_ctype: en_US.UTF-8
    template: template0
    grants:
      - user: grant_test_user
        privileges: [CONNECT]
      - user: grant_readonly_user
        privileges: [CONNECT]
    schemas:
      - name: grant_test_schema
        owner: grant_test_user
        grants:
          - user: grant_test_user
            privileges: [USAGE, CREATE]
            table_privileges: [SELECT, INSERT, UPDATE, DELETE]
            sequence_privileges: [USAGE, SELECT, UPDATE]
            function_privileges: [EXECUTE]
            default_privileges: [SELECT, INSERT, UPDATE, DELETE]
          - role: grant_readonly_role
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

	// Connect to the test database as the owner
	testDbConfig, err := pgx.ParseConfig(dbURL)
	require.NoError(t, err)
	testDbConfig.Database = "grant_test_db"
	testDbConfig.User = "grant_test_user"
	testDbConfig.Password = "testpass123"
	
	ownerConn, err := pgx.ConnectConfig(ctx, testDbConfig)
	require.NoError(t, err)
	defer ownerConn.Close(ctx)

	// Create test objects to verify grants
	_, err = ownerConn.Exec(ctx, `
		CREATE TABLE grant_test_schema.test_table (id serial PRIMARY KEY, name text);
		CREATE SEQUENCE grant_test_schema.test_sequence;
		CREATE FUNCTION grant_test_schema.test_function() RETURNS text AS $$ 
			BEGIN RETURN 'Hello, World!'; END; 
		$$ LANGUAGE plpgsql;
		INSERT INTO grant_test_schema.test_table (name) VALUES ('test1'), ('test2');
	`)
	require.NoError(t, err)

	// Connect as the readonly user
	readonlyDbConfig, err := pgx.ParseConfig(dbURL)
	require.NoError(t, err)
	readonlyDbConfig.Database = "grant_test_db"
	readonlyDbConfig.User = "grant_readonly_user"
	readonlyDbConfig.Password = "testpass123"
	
	readonlyConn, err := pgx.ConnectConfig(ctx, readonlyDbConfig)
	require.NoError(t, err)
	defer readonlyConn.Close(ctx)

	// Test 1: Table privileges - readonly user should be able to SELECT but not INSERT
	var count int
	err = readonlyConn.QueryRow(ctx, "SELECT COUNT(*) FROM grant_test_schema.test_table").Scan(&count)
	assert.NoError(t, err, "Readonly user should be able to SELECT")
	assert.Equal(t, 2, count, "Should see 2 rows in the table")

	_, err = readonlyConn.Exec(ctx, "INSERT INTO grant_test_schema.test_table (name) VALUES ('test3')")
	assert.Error(t, err, "Readonly user should not be able to INSERT")

	// Test 2: Sequence privileges - readonly user should be able to SELECT but not UPDATE
	var lastVal int
	err = readonlyConn.QueryRow(ctx, "SELECT last_value FROM grant_test_schema.test_sequence").Scan(&lastVal)
	assert.NoError(t, err, "Readonly user should be able to SELECT from sequence")

	_, err = readonlyConn.Exec(ctx, "SELECT nextval('grant_test_schema.test_sequence')")
	assert.NoError(t, err, "Readonly user should be able to use USAGE on sequence")

	_, err = readonlyConn.Exec(ctx, "ALTER SEQUENCE grant_test_schema.test_sequence RESTART WITH 100")
	assert.Error(t, err, "Readonly user should not be able to ALTER sequence")

	// Test 3: Function privileges - readonly user should be able to EXECUTE
	var result string
	err = readonlyConn.QueryRow(ctx, "SELECT grant_test_schema.test_function()").Scan(&result)
	assert.NoError(t, err, "Readonly user should be able to EXECUTE function")
	assert.Equal(t, "Hello, World!", result)

	// Test 4: Default privileges - create a new table as owner and verify readonly can SELECT
	_, err = ownerConn.Exec(ctx, "CREATE TABLE grant_test_schema.another_table (id int, value text)")
	require.NoError(t, err)
	
	_, err = ownerConn.Exec(ctx, "INSERT INTO grant_test_schema.another_table VALUES (1, 'test')")
	require.NoError(t, err)

	var val string
	err = readonlyConn.QueryRow(ctx, "SELECT value FROM grant_test_schema.another_table WHERE id = 1").Scan(&val)
	assert.NoError(t, err, "Readonly user should be able to SELECT from new table due to default privileges")
	assert.Equal(t, "test", val)

	_, err = readonlyConn.Exec(ctx, "INSERT INTO grant_test_schema.another_table VALUES (2, 'test2')")
	assert.Error(t, err, "Readonly user should not be able to INSERT into new table")

	// Clean up
	conn.Exec(ctx, "DROP DATABASE grant_test_db")
	conn.Exec(ctx, "DROP ROLE grant_test_user")
	conn.Exec(ctx, "DROP ROLE grant_readonly_user")
	conn.Exec(ctx, "DROP ROLE grant_readonly_role")
}
