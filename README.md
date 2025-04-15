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