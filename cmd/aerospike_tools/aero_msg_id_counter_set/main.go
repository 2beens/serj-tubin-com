package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/2beens/serjtubincom/internal/config"
	"github.com/2beens/serjtubincom/tools"
)

// set / fix the id counter for visitor board messages
func main() {
	fmt.Println("starting aerospike id counter for visitor board messages fix/set ...")

	env := flag.String("env", "development", "environment [prod | production | dev | development | ddev | dockerdev ]")
	configPath := flag.String("config", "./config.toml", "path for the TOML config file")
	flag.Parse()

	cfg, err := config.Load(*env, *configPath)
	if err != nil {
		panic(err)
	}

	if err := tools.FixAerospikeMessageIdCounter(cfg.AeroNamespace, cfg.AeroMessagesSet, cfg.AeroHost, cfg.AeroPort); err != nil {
		fmt.Printf("aero message id counter fix / set failed: %s\n", err)
		os.Exit(1)
	}

	fmt.Println("\naero message id counter fix / set completed")
}
