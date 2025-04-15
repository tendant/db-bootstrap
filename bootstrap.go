package bootstrap

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/jackc/pgx/v5"
	"gopkg.in/yaml.v3"
)

var DefaultYAML []byte

type User struct {
	Name        string   `yaml:"name"`
	PasswordEnv string   `yaml:"password_env"`
	Password    string   // populated at runtime
	CanLogin    bool     `yaml:"can_login"`
	OwnsSchemas []string `yaml:"owns_schemas"`
	Roles       []string `yaml:"roles"`
}

type SchemaGrant struct {
	User       string   `yaml:"user"`
	Privileges []string `yaml:"privileges"`
}

type Schema struct {
	Name   string        `yaml:"name"`
	Owner  string        `yaml:"owner"`
	Grants []SchemaGrant `yaml:"grants"`
}

type DatabaseGrant struct {
	User       string   `yaml:"user"`
	Privileges []string `yaml:"privileges"`
}

type Database struct {
	Name       string          `yaml:"name"`
	Owner      string          `yaml:"owner"`
	Encoding   string          `yaml:"encoding"`
	LcCollate  string          `yaml:"lc_collate"`
	LcCtype    string          `yaml:"lc_ctype"`
	Template   string          `yaml:"template"`
	Extensions []string        `yaml:"extensions"`
	Grants     []DatabaseGrant `yaml:"grants"`
	Schemas    []Schema        `yaml:"schemas"`
}

type Config struct {
	Users     []User     `yaml:"users"`
	Databases []Database `yaml:"databases"`
}

func getEnvBool(key string) bool {
	v := os.Getenv(key)
	return strings.ToLower(v) == "true" || v == "1" || v == "yes"
}

// Note: LoadAndRenderSQL function has been removed as we now handle extensions at the database level

// createDatabases creates databases directly through the database connection
func createDatabases(ctx context.Context, dbURL string, databases []Database) error {
	if len(databases) == 0 {
		return nil
	}

	// Connect to the default database
	slog.Info("Connecting to database to create databases")
	conn, err := pgx.Connect(ctx, dbURL)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer conn.Close(ctx)

	// Create each database
	for _, db := range databases {
		// Check if database exists
		var exists bool
		err := conn.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)", db.Name).Scan(&exists)
		if err != nil {
			return fmt.Errorf("failed to check if database exists: %w", err)
		}

		if !exists {
			// Build CREATE DATABASE command
			createCmd := fmt.Sprintf("CREATE DATABASE %s", db.Name)

			if db.Owner != "" {
				createCmd += fmt.Sprintf(" OWNER %s", db.Owner)
			}
			if db.Encoding != "" {
				createCmd += fmt.Sprintf(" ENCODING '%s'", db.Encoding)
			}
			if db.LcCollate != "" {
				createCmd += fmt.Sprintf(" LC_COLLATE '%s'", db.LcCollate)
			}
			if db.LcCtype != "" {
				createCmd += fmt.Sprintf(" LC_CTYPE '%s'", db.LcCtype)
			}
			if db.Template != "" {
				createCmd += fmt.Sprintf(" TEMPLATE %s", db.Template)
			}

			slog.Info("Creating database", "name", db.Name)
			// Execute the CREATE DATABASE command
			_, err = conn.Exec(ctx, createCmd)
			if err != nil {
				return fmt.Errorf("failed to create database %s: %w", db.Name, err)
			}
			slog.Info("Created database", "name", db.Name)
		} else {
			slog.Info("Database already exists", "name", db.Name)
		}

		// Apply grants
		for _, grant := range db.Grants {
			privileges := strings.Join(grant.Privileges, ", ")
			grantCmd := fmt.Sprintf("GRANT %s ON DATABASE %s TO %s", privileges, db.Name, grant.User)
			slog.Info("Applying grant", "database", db.Name, "user", grant.User, "privileges", privileges)
			_, err = conn.Exec(ctx, grantCmd)
			if err != nil {
				return fmt.Errorf("failed to grant privileges on database %s: %w", db.Name, err)
			}
		}
	}

	return nil
}

// createUsers creates users directly through the database connection
func createUsers(ctx context.Context, dbURL string, users []User) error {
	if len(users) == 0 {
		return nil
	}

	// Connect to the default database
	slog.Info("Connecting to database to create users")
	conn, err := pgx.Connect(ctx, dbURL)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer conn.Close(ctx)

	// Create each user
	for _, user := range users {
		// Check if user exists
		var exists bool
		err := conn.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM pg_roles WHERE rolname = $1)", user.Name).Scan(&exists)
		if err != nil {
			return fmt.Errorf("failed to check if user exists: %w", err)
		}

		if !exists {
			// Build CREATE ROLE command
			createCmd := fmt.Sprintf("CREATE ROLE %s", user.Name)
			if user.CanLogin {
				createCmd += fmt.Sprintf(" WITH LOGIN PASSWORD '%s'", user.Password)
			}

			slog.Info("Creating user", "name", user.Name)
			// Execute the CREATE ROLE command
			_, err = conn.Exec(ctx, createCmd)
			if err != nil {
				return fmt.Errorf("failed to create user %s: %w", user.Name, err)
			}
			slog.Info("Created user", "name", user.Name)
		} else {
			slog.Info("User already exists", "name", user.Name)
		}

		// Apply roles
		for _, role := range user.Roles {
			grantCmd := fmt.Sprintf("GRANT %s TO %s", role, user.Name)
			slog.Info("Applying role grant", "user", user.Name, "role", role)
			_, err = conn.Exec(ctx, grantCmd)
			if err != nil {
				return fmt.Errorf("failed to grant role %s to user %s: %w", role, user.Name, err)
			}
		}
	}

	return nil
}

// createSchemas creates schemas within a database
func createSchemas(ctx context.Context, conn *pgx.Conn, schemas []Schema) error {
	if len(schemas) == 0 {
		return nil
	}

	// Create each schema
	for _, schema := range schemas {
		// Check if schema exists
		var exists bool
		err := conn.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM information_schema.schemata WHERE schema_name = $1)", schema.Name).Scan(&exists)
		if err != nil {
			return fmt.Errorf("failed to check if schema exists: %w", err)
		}

		if !exists {
			// Build CREATE SCHEMA command
			createCmd := fmt.Sprintf("CREATE SCHEMA %s AUTHORIZATION %s", schema.Name, schema.Owner)

			slog.Info("Creating schema", "name", schema.Name, "owner", schema.Owner)
			// Execute the CREATE SCHEMA command
			_, err = conn.Exec(ctx, createCmd)
			if err != nil {
				return fmt.Errorf("failed to create schema %s: %w", schema.Name, err)
			}
			slog.Info("Created schema", "name", schema.Name)
		} else {
			slog.Info("Schema already exists", "name", schema.Name)
		}

		// Apply grants
		for _, grant := range schema.Grants {
			privileges := strings.Join(grant.Privileges, ", ")
			grantCmd := fmt.Sprintf("GRANT %s ON SCHEMA %s TO %s", privileges, schema.Name, grant.User)
			slog.Info("Applying schema grant", "schema", schema.Name, "user", grant.User, "privileges", privileges)
			_, err = conn.Exec(ctx, grantCmd)
			if err != nil {
				return fmt.Errorf("failed to grant privileges on schema %s: %w", schema.Name, err)
			}
		}
	}

	return nil
}

// createExtensions creates extensions within a database
func createExtensions(ctx context.Context, conn *pgx.Conn, extensions []string) error {
	if len(extensions) == 0 {
		return nil
	}

	// Create each extension
	for _, extension := range extensions {
		// Check if extension exists
		var exists bool
		err := conn.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM pg_extension WHERE extname = $1)", extension).Scan(&exists)
		if err != nil {
			return fmt.Errorf("failed to check if extension exists: %w", err)
		}

		if !exists {
			// Build CREATE EXTENSION command
			createCmd := fmt.Sprintf("CREATE EXTENSION IF NOT EXISTS %s", extension)

			slog.Info("Creating extension", "name", extension)
			// Execute the CREATE EXTENSION command
			_, err = conn.Exec(ctx, createCmd)
			if err != nil {
				return fmt.Errorf("failed to create extension %s: %w", extension, err)
			}
			slog.Info("Created extension", "name", extension)
		} else {
			slog.Info("Extension already exists", "name", extension)
		}
	}

	return nil
}

func BootstrapDatabase(yamlData []byte) error {
	// Parse the YAML configuration
	slog.Info("Parsing YAML configuration")
	var config Config
	if err := yaml.Unmarshal(yamlData, &config); err != nil {
		return fmt.Errorf("failed to unmarshal yaml: %w", err)
	}

	// Set passwords from environment variables
	slog.Info("Setting passwords from environment variables")
	for i := range config.Users {
		if config.Users[i].PasswordEnv != "" {
			pw := os.Getenv(config.Users[i].PasswordEnv)
			if pw == "" {
				return fmt.Errorf("missing env var: %s for user %s", config.Users[i].PasswordEnv, config.Users[i].Name)
			}
			config.Users[i].Password = pw
		}
	}

	if outputPath := os.Getenv("BOOTSTRAP_OUTPUT_PATH"); outputPath != "" {
		slog.Info("Output path specified but no longer used for SQL generation")
	}

	if getEnvBool("BOOTSTRAP_RENDER_ONLY") || getEnvBool("BOOTSTRAP_DRY_RUN") {
		slog.Info("DRY RUN MODE - No changes will be made")
		return nil
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return fmt.Errorf("DATABASE_URL must be set")
	}

	ctx := context.Background()

	// 1. Create users first
	slog.Info("Starting user creation")
	if err := createUsers(ctx, dbURL, config.Users); err != nil {
		return err
	}

	// 2. Create databases
	if len(config.Databases) > 0 {
		slog.Info("Starting database creation")
		if err := createDatabases(ctx, dbURL, config.Databases); err != nil {
			return err
		}
	}

	// 3. Create extensions and schemas within each database
	for _, db := range config.Databases {
		slog.Info("Processing database", "database", db.Name)

		// Parse the original URL
		dbConfig, err := pgx.ParseConfig(dbURL)
		if err != nil {
			return fmt.Errorf("failed to parse database URL: %w", err)
		}

		// Update the database name
		dbConfig.Database = db.Name

		// Connect to the specific database
		slog.Info("Connecting to database", "database", db.Name)
		conn, err := pgx.ConnectConfig(ctx, dbConfig)
		if err != nil {
			return fmt.Errorf("failed to connect to database %s: %w", db.Name, err)
		}

		// Create extensions for this database
		if len(db.Extensions) > 0 {
			slog.Info("Creating extensions", "database", db.Name, "extensions", db.Extensions)
			if err := createExtensions(ctx, conn, db.Extensions); err != nil {
				conn.Close(ctx)
				return err
			}
		}

		// Create schemas for this database
		if len(db.Schemas) > 0 {
			slog.Info("Creating schemas", "database", db.Name)
			if err := createSchemas(ctx, conn, db.Schemas); err != nil {
				conn.Close(ctx)
				return err
			}
		}

		conn.Close(ctx)
	}

	slog.Info("Bootstrap executed successfully")
	return nil
}
