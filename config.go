package main

import (
	"encoding/json"
	"fmt"
	"os"
)

// Configuration structure for network_health_exporter
type Configuration struct {
	Targets         []string
	IntervalSeconds int
	TimeoutSeconds  int
	Port            int
}

// ReadConfig read the configuration file at loc
func ReadConfig(loc string) Configuration {
	fmt.Printf("Reading %s ...\n", loc)
	file, _ := os.Open(loc)
	decoder := json.NewDecoder(file)
	config := Configuration{}
	err := decoder.Decode(&config)
	if err != nil {
		fmt.Println("error reading config", err)
		os.Exit(1)
	}
	return config
}
