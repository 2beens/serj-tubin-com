// Package main runs the gymstats MCP server over stdio (for local Cursor use).
// The same MCP server is also mounted on the main backend at /mcp over HTTP,
// so you can use either: stdio (this cmd) or the backend URL (no extra deploy).
package main

import (
	"context"
	"flag"
	"log"

	"github.com/2beens/serjtubincom/internal/config"
	"github.com/2beens/serjtubincom/internal/db"
	"github.com/2beens/serjtubincom/internal/gymstats/exercises"
	gymstatsmcp "github.com/2beens/serjtubincom/internal/gymstats/mcp"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	env := flag.String("env", "development", "environment [prod | production | dev | development | ddev | dockerdev]")
	configPath := flag.String("config", "./config.toml", "path to TOML config file")
	flag.Parse()

	cfg, err := config.Load(*env, *configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	ctx := context.Background()
	dbPool, err := db.NewDBPool(ctx, db.NewDBPoolParams{
		DBHost:         cfg.PostgresHost,
		DBPort:         cfg.PostgresPort,
		DBName:         cfg.PostgresDBName,
		TracingEnabled: false,
	})
	if err != nil {
		log.Fatalf("db pool: %v", err)
	}
	defer dbPool.Close()

	exercisesRepo := exercises.NewRepo(dbPool)
	server := gymstatsmcp.NewServer(dbPool, exercisesRepo)

	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatal(err)
	}
}
