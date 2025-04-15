# pg-bootstrap

A CLI tool for bootstrapping PostgreSQL databases, users, schemas, and extensions from a YAML configuration. Powered by Golang and [kong](https://github.com/alecthomas/kong) CLI library.

## Features

- Create users with login privileges
- Set user roles and ownerships
- Bootstrap databases with custom encoding, collation, and templates
- Create schemas with specific grants
- Install database extensions
- Manage grants at both database and schema levels

## Installation

Clone the repo and build the CLI:

```bash
git clone https://github.com/tendant/db-bootstrap.git
cd db-bootstrap
go build -o db-bootstrap ./cmd/bootstrap


## How to run

1. **Set environment variables**:

   ```bash
   export DATABASE_URL=postgres://postgres:pwd@localhost:5432/postgres
   export TEST_USER_PASSWORD=pass123
   
2. Create your config file (bootstrap.yaml):

See the Sample Config section for a working example.

3. Run the CLI:

    go run ./cmd/bootstrap run --config-path=samples/bootstrap.yaml