# dbstrap

A CLI tool for bootstrapping PostgreSQL databases, users, schemas, and extensions from a YAML configuration. Powered by Golang and [kong](https://github.com/alecthomas/kong) CLI library.

## Features

- Create users with login privileges
- Set user roles and ownerships
- Bootstrap databases with custom encoding, collation, and templates
- Create schemas with specific grants to users and roles, including table, sequence, function, and default privileges
- Install database extensions
- Manage grants at both database and schema levels

## Installation

Clone the repo and build the CLI:

```bash
go install github.com/tendant/dbstrap/cmd/dbstrap@latest
```

## Run from binary

```bash
export DATABASE_URL=postgres://postgres:pwd@localhost:5432/postgres
export TEST_USER_PASSWORD=pass123
dbstrap run --config=samples/bootstrap.yaml
```

## How to run from source

1. **Set environment variables**:

   ```bash
   export DATABASE_URL=postgres://postgres:pwd@localhost:5432/postgres
   export TEST_USER_PASSWORD=pass123
   
2. Create your config file (bootstrap.yaml):

See the Sample Config section for a working example.

3. Run the CLI:

    go run ./cmd/dbstrap run --config=samples/bootstrap.yaml

## Sample Config

Here's an example configuration that demonstrates the main features:

```yaml
users:
  - name: test_user
    password_env: TEST_USER_PASSWORD
    can_login: true
    owns_schemas:
      - public
    roles: []
  - name: read_only_user
    password_env: TEST_USER_PASSWORD
    can_login: true
    owns_schemas: []
    roles: [readonly_role]

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
            sequence_privileges: [USAGE, SELECT, UPDATE]
            function_privileges: [EXECUTE]
            default_privileges: [SELECT, INSERT, UPDATE, DELETE]
          - role: readonly_role
            privileges: [USAGE]
            table_privileges: [SELECT]
            sequence_privileges: [USAGE, SELECT]
            function_privileges: [EXECUTE]
            default_privileges: [SELECT]
      - name: analytics
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
```

In this example, we're creating:
- Two users: `test_user` (owner) and `read_only_user` (with readonly access)
- A database named `test_db` with the UUID extension
- Two schemas: `public` and `analytics`
- Grants at both database and schema levels, including role-based permissions
- Comprehensive object privileges within schemas:
  - Table privileges for all existing tables
  - Sequence privileges for all sequences (important for auto-incrementing columns)
  - Function privileges for all stored procedures and functions
  - Default privileges for future objects created in the schema