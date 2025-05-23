package main

import (
	"log"
	"os"

	"github.com/alecthomas/kong"
	"github.com/tendant/dbstrap"
)

var CLI struct {
	Run struct {
		Config string `help:"Path to YAML bootstrap config" default:"bootstrap.yaml"`
	} `cmd:"" help:"Run the dbstrap process"`
}

func main() {
	kctx := kong.Parse(&CLI,
		kong.Name("dbstrap"),
		kong.Description("Database bootstrap CLI tool."),
		kong.UsageOnError(),
	)

	switch kctx.Command() {
	case "run":
		yamlFile, err := os.ReadFile(CLI.Run.Config)
		if err != nil {
			log.Fatalf("Failed to read config file: %v", err)
		}

		if err := dbstrap.BootstrapDatabase(yamlFile); err != nil {
			log.Fatalf("Failed to bootstrap database: %v", err)
		}
	default:
		log.Fatalf("Unknown command: %s", kctx.Command())
	}
}
