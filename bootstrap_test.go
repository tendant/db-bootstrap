package dbstrap

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// Helper function for tests to parse YAML config
func parseConfig(yamlData []byte, config *Config) error {
	return yaml.Unmarshal(yamlData, config)
}

// TestParseConfig tests the YAML parsing functionality
func TestParseConfig(t *testing.T) {
	yamlData := []byte(`
users:
  - name: test_user
    password_env: TEST_USER_PASSWORD
    can_login: true
    owns_schemas:
      - public
    roles: []

databases:
  - name: test_db
    owner: test_user
    encoding: UTF8
    lc_collate: en_US.UTF-8
    lc_ctype: en_US.UTF-8
    template: template0
    extensions:
      - "uuid-ossp"
    grants:
      - user: test_user
        privileges: [CONNECT]
    schemas:
      - name: public
        owner: test_user
        grants:
          - user: test_user
            privileges: [USAGE, CREATE]
            table_privileges: [SELECT, INSERT, UPDATE, DELETE]
          - role: readonly_role
            privileges: [USAGE]
            table_privileges: [SELECT]
`)

	var config Config
	err := parseConfig(yamlData, &config)
	require.NoError(t, err)

	// Verify users
	require.Len(t, config.Users, 1)
	assert.Equal(t, "test_user", config.Users[0].Name)
	assert.Equal(t, "TEST_USER_PASSWORD", config.Users[0].PasswordEnv)
	assert.True(t, config.Users[0].CanLogin)
	assert.Contains(t, config.Users[0].OwnsSchemas, "public")

	// Verify databases
	require.Len(t, config.Databases, 1)
	db := config.Databases[0]
	assert.Equal(t, "test_db", db.Name)
	assert.Equal(t, "test_user", db.Owner)
	assert.Equal(t, "UTF8", db.Encoding)
	assert.Equal(t, "en_US.UTF-8", db.LcCollate)
	assert.Equal(t, "en_US.UTF-8", db.LcCtype)
	assert.Equal(t, "template0", db.Template)
	assert.Contains(t, db.Extensions, "uuid-ossp")

	// Verify grants
	require.Len(t, db.Grants, 1)
	assert.Equal(t, "test_user", db.Grants[0].User)
	assert.Contains(t, db.Grants[0].Privileges, "CONNECT")

	// Verify schemas
	require.Len(t, db.Schemas, 1)
	schema := db.Schemas[0]
	assert.Equal(t, "public", schema.Name)
	assert.Equal(t, "test_user", schema.Owner)

	// Verify schema grants
	require.Len(t, schema.Grants, 2)
	
	// User grant
	userGrant := schema.Grants[0]
	assert.Equal(t, "test_user", userGrant.User)
	assert.Contains(t, userGrant.Privileges, "USAGE")
	assert.Contains(t, userGrant.Privileges, "CREATE")
	assert.Contains(t, userGrant.TablePrivileges, "SELECT")
	assert.Contains(t, userGrant.TablePrivileges, "INSERT")
	
	// Role grant
	roleGrant := schema.Grants[1]
	assert.Equal(t, "readonly_role", roleGrant.Role)
	assert.Contains(t, roleGrant.Privileges, "USAGE")
	assert.Contains(t, roleGrant.TablePrivileges, "SELECT")
}
