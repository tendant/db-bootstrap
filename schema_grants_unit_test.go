package dbstrap

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSchemaGrantConfig tests the parsing of schema grant configurations
// with all the new features: role-based grants, table privileges, sequence privileges,
// function privileges, and default privileges
func TestSchemaGrantConfig(t *testing.T) {
	yamlData := []byte(`
users:
  - name: test_user
    password_env: TEST_USER_PASSWORD
    can_login: true
    owns_schemas:
      - test_schema
    roles: []
  - name: readonly_user
    password_env: TEST_USER_PASSWORD
    can_login: true
    owns_schemas: []
    roles: [readonly_role]

databases:
  - name: test_db
    owner: test_user
    schemas:
      - name: test_schema
        owner: test_user
        grants:
          - user: test_user
            privileges: [USAGE, CREATE]
            table_privileges: [SELECT, INSERT, UPDATE, DELETE]
            sequence_privileges: [USAGE, SELECT, UPDATE]
            function_privileges: [EXECUTE]
            default_privileges: [SELECT, INSERT, UPDATE, DELETE]
          - role: readonly_role
            privileges: [USAGE]
            table_privileges: [SELECT]
            sequence_privileges: [USAGE, SELECT]
            function_privileges: [EXECUTE]
            default_privileges: [SELECT]
`)

	var config Config
	err := parseConfig(yamlData, &config)
	require.NoError(t, err)

	// Verify users
	require.Len(t, config.Users, 2)
	assert.Equal(t, "test_user", config.Users[0].Name)
	assert.Equal(t, "readonly_user", config.Users[1].Name)
	assert.Contains(t, config.Users[1].Roles, "readonly_role")

	// Verify databases and schemas
	require.Len(t, config.Databases, 1)
	db := config.Databases[0]
	assert.Equal(t, "test_db", db.Name)
	
	// Verify schemas
	require.Len(t, db.Schemas, 1)
	schema := db.Schemas[0]
	assert.Equal(t, "test_schema", schema.Name)
	assert.Equal(t, "test_user", schema.Owner)

	// Verify schema grants
	require.Len(t, schema.Grants, 2)
	
	// User grant with full privileges
	userGrant := schema.Grants[0]
	assert.Equal(t, "test_user", userGrant.User)
	assert.Empty(t, userGrant.Role)
	assert.ElementsMatch(t, []string{"USAGE", "CREATE"}, userGrant.Privileges)
	assert.ElementsMatch(t, []string{"SELECT", "INSERT", "UPDATE", "DELETE"}, userGrant.TablePrivileges)
	assert.ElementsMatch(t, []string{"USAGE", "SELECT", "UPDATE"}, userGrant.SequencePrivileges)
	assert.ElementsMatch(t, []string{"EXECUTE"}, userGrant.FunctionPrivileges)
	assert.ElementsMatch(t, []string{"SELECT", "INSERT", "UPDATE", "DELETE"}, userGrant.DefaultPrivileges)
	
	// Role grant with readonly privileges
	roleGrant := schema.Grants[1]
	assert.Empty(t, roleGrant.User)
	assert.Equal(t, "readonly_role", roleGrant.Role)
	assert.ElementsMatch(t, []string{"USAGE"}, roleGrant.Privileges)
	assert.ElementsMatch(t, []string{"SELECT"}, roleGrant.TablePrivileges)
	assert.ElementsMatch(t, []string{"USAGE", "SELECT"}, roleGrant.SequencePrivileges)
	assert.ElementsMatch(t, []string{"EXECUTE"}, roleGrant.FunctionPrivileges)
	assert.ElementsMatch(t, []string{"SELECT"}, roleGrant.DefaultPrivileges)
}

// TestSchemaGrantValidation tests validation of schema grants
func TestSchemaGrantValidation(t *testing.T) {
	// Test case: Missing both user and role
	invalidGrant := SchemaGrant{
		Privileges:     []string{"USAGE"},
		TablePrivileges: []string{"SELECT"},
	}
	
	// Create a test schema with the invalid grant
	schema := Schema{
		Name:   "test_schema",
		Owner:  "test_user",
		Grants: []SchemaGrant{invalidGrant},
	}
	
	// Create a test database with the schema
	db := Database{
		Name:    "test_db",
		Owner:   "test_user",
		Schemas: []Schema{schema},
	}
	
	// Create a test config with the database
	config := Config{
		Databases: []Database{db},
	}
	
	// Verify that createSchemas would fail with this config
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, "postgres://postgres:pwd@localhost:5432/postgres")
	if err == nil {
		defer conn.Close(ctx)
		err = createSchemas(ctx, conn, config.Databases[0].Schemas)
		assert.Error(t, err, "createSchemas should fail with invalid grant")
		assert.Contains(t, err.Error(), "schema grant must specify either user or role")
	} else {
		t.Skip("Skipping validation test as database connection failed")
	}
}
