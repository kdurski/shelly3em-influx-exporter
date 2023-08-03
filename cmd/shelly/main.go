package main

import (
	"flag"
	"github.com/joho/godotenv"
	"log"
	"os"
	"time"
)

type EnergyPoint struct {
	Phase     string
	Consumed  float64
	Returned  float64
	Timestamp time.Time
}

func main() {
	// accept dry run option using the flag library
	dryRun := flag.Bool("dry-run", false, "dry run")
	flag.Parse()

	if _, err := os.Stat(".env.local"); !os.IsNotExist(err) {
		if err := godotenv.Load(".env.local"); err != nil {
			log.Fatal("Error loading .env.local file: ", err)
		}
	}

	if err := godotenv.Load(".env"); err != nil {
		log.Fatal("Error loading .env file: ", err)
	}

	importFromCsv(dryRun)
}
