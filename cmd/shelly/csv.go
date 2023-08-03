package main

import (
	"encoding/csv"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/influxdata/influxdb-client-go/v2/api/write"
	"io"
	"log"
	"net/http"
	"os"
)

func importFromCsv(dryRun *bool) {
	if *dryRun {
		log.Printf("Dry run enabled...")
		log.Printf("CSV A: %s", os.Getenv("CSV_A"))
		log.Printf("CSV B: %s", os.Getenv("CSV_B"))
		log.Printf("CSV C: %s", os.Getenv("CSV_C"))

	} else {
		log.Printf("Importing CSVs...")
		var InfluxDbToken = os.Getenv("INFLUXDB_TOKEN")
		var InfluxDbUrl = os.Getenv("INFLUXDB_URL")
		var InfluxOrg = os.Getenv("INFLUXDB_ORG")
		var InfluxBucket = os.Getenv("INFLUXDB_BUCKET")

		log.Printf("InfluxDb Url: %s, Org: %s, Bucket: %s", InfluxDbUrl, InfluxOrg, InfluxBucket)

		client := influxdb2.NewClient(InfluxDbUrl, InfluxDbToken)
		writeAPI := client.WriteAPI(InfluxOrg, InfluxBucket)

		processCSV(writeAPI, os.Getenv("CSV_A"), "A")
		processCSV(writeAPI, os.Getenv("CSV_B"), "B")
		processCSV(writeAPI, os.Getenv("CSV_C"), "C")
	}

	log.Print("Done!")
}

func writePoint(writeAPI api.WriteAPI, ep EnergyPoint) {
	point := write.NewPoint(
		"energy",
		map[string]string{"phase": ep.Phase},
		map[string]any{
			"consumed": ep.Consumed,
			"returned": ep.Returned,
		},
		ep.Timestamp,
	)

	writeAPI.WritePoint(point)
}

func downloadCsv(url string) string {
	tempFile := os.TempDir() + "/shelly-" + randomString(10) + ".csv"
	log.Printf("Downloading csv %s to %s\n", url, tempFile)
	out, err := os.Create(tempFile)
	if err != nil {
		log.Panic(err)
	}

	resp, err := http.Get(url)
	if err != nil {
		log.Panic(err)
	}
	defer resp.Body.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		log.Panic(err)
	}

	log.Printf("Downloaded csv %s to %s\n", url, tempFile)

	return tempFile
}

func processCSV(writeAPI api.WriteAPI, url, phase string) {
	path := downloadCsv(url)
	f, err := os.Open(path)
	if err != nil {
		log.Panic("Error: ", err)
	}
	defer f.Close()

	csvReader := csv.NewReader(f)
	counter := 0

	// parse header
	if header, err := csvReader.Read(); err != nil {
		log.Panic("Error: ", err)
	} else {
		if header[0] == "Another file transfer is in progress!" {
			log.Fatalf("Another file transfer is in progress: %s", url)
		}
	}

	// read all lines one by one
	for {
		rec, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Panic(err)
		}

		ep := EnergyPoint{
			Phase:     phase,
			Consumed:  mustParseFloat(rec[1]),
			Returned:  mustParseFloat(rec[2]),
			Timestamp: mustParseTime(rec[0]),
		}

		writePoint(writeAPI, ep)
		counter++
	}

	writeAPI.Flush()
	log.Printf("Points written for csv %s and phase %s: %d\n", path, phase, counter)
}
