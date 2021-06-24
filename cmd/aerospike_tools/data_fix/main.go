package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/2beens/serjtubincom/internal/config"
	"github.com/2beens/serjtubincom/tools"
)

// run aerospike sql data fixing / migration
func main() {
	fmt.Println("starting aerospike data fix/migration tools ...")

	env := flag.String("env", "development", "environment [prod | production | dev | development]")
	configPath := flag.String("config", "./config.toml", "path for the TOML config file")
	flag.Parse()

	cfg, err := config.Load(*env, *configPath)
	if err != nil {
		panic(err)
	}

	if err := tools.FixAerospikeData(cfg.AeroNamespace, cfg.AeroMessagesSet, cfg.AeroHost, cfg.AeroPort); err != nil {
		fmt.Printf("aero data fix failed: %s\n", err)
		os.Exit(1)
	}

	fmt.Println("\naero data fix completed")
}
