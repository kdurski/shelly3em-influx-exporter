package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
)

type EnergyPoint struct {
	Phase     string
	Consumed  float64
	Returned  float64
	Timestamp time.Time
}

func main() {
	flag.Usage = usage

	dryRun := flag.Bool("dry-run", false, "dry run")
	checkConnectivity := flag.Bool("check-connectivity", false, "check configured TCP connectivity and exit")
	flag.Parse()

	if _, err := os.Stat(".env.local"); !os.IsNotExist(err) {
		if err := godotenv.Load(".env.local"); err != nil {
			log.Fatal("Error loading .env.local file: ", err)
		}
	}

	if err := godotenv.Load(".env"); err != nil {
		log.Fatal("Error loading .env file: ", err)
	}

	if err := run(*dryRun, *checkConnectivity); err != nil {
		log.Fatal(err)
	}
}

func run(dryRun, checkConnectivity bool) error {
	if checkConnectivity {
		return checkConfiguredConnectivity()
	}

	return importFromCsv(dryRun)
}

func usage() {
	out := flag.CommandLine.Output()
	fmt.Fprintf(out, "Usage: %s [flags]\n\n", os.Args[0])
	fmt.Fprintln(out, "Exports Shelly 3EM CSV data to InfluxDB.")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Flags:")
	flag.PrintDefaults()
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Examples:")
	fmt.Fprintf(out, "  %s --help\n", os.Args[0])
	fmt.Fprintf(out, "  %s --dry-run\n", os.Args[0])
	fmt.Fprintf(out, "  %s --check-connectivity\n", os.Args[0])
}
