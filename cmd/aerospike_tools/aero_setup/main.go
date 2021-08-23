package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/2beens/serjtubincom/internal/config"
	"github.com/2beens/serjtubincom/tools"
)

// run aerospike sql setup
func main() {
	fmt.Println("starting aerospike sql setup ...")

	env := flag.String("env", "development", "environment [prod | production | dev | development | ddev | dockerdev ]")
	configPath := flag.String("config", "./config.toml", "path for the TOML config file")
	flag.Parse()

	cfg, err := config.Load(*env, *configPath)
	if err != nil {
		panic(err)
	}

	if err := tools.SetupAeroDb(cfg.AeroNamespace, cfg.AeroMessagesSet, cfg.AeroHost, cfg.AeroPort); err != nil {
		fmt.Printf("aero setup failed: %s\n", err)
		os.Exit(1)
	}

	fmt.Println("\naero setup completed")
}
