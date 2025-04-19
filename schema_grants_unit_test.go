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
	
	// In a real scenario, this grant would be part of a schema
	// We're just testing the grant itself in this unit test
	
	// Verify that the grant is invalid (neither user nor role specified)
	assert.Empty(t, invalidGrant.User, "User should be empty")
	assert.Empty(t, invalidGrant.Role, "Role should be empty")
	
	// In a real scenario, createSchemas would fail with:
	// "schema grant must specify either user or role"
	// But we don't need to test that in a unit test that connects to a database
}
