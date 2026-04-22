package main

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/influxdata/influxdb-client-go/v2/api/write"
)

var downloadClient = &http.Client{Timeout: 300 * time.Second}

func importFromCsv(dryRun bool) error {
	if dryRun {
		log.Printf("Dry run enabled...")
		log.Printf("CSV A: %s", os.Getenv("CSV_A"))
		log.Printf("CSV B: %s", os.Getenv("CSV_B"))
		log.Printf("CSV C: %s", os.Getenv("CSV_C"))
		log.Print("Done!")
		return nil
	}

	log.Printf("Importing CSVs...")
	influxDBToken := os.Getenv("INFLUXDB_TOKEN")
	influxDBURL := os.Getenv("INFLUXDB_URL")
	influxOrg := os.Getenv("INFLUXDB_ORG")
	influxBucket := os.Getenv("INFLUXDB_BUCKET")

	log.Printf("InfluxDb Url: %s, Org: %s, Bucket: %s", influxDBURL, influxOrg, influxBucket)
	if err := checkConfiguredConnectivity(); err != nil {
		return err
	}

	client := influxdb2.NewClient(influxDBURL, influxDBToken)
	writeAPI := client.WriteAPI(influxOrg, influxBucket)
	asyncWriteErrors := collectAsyncWriteErrors(writeAPI.Errors())

	if err := processCSV(writeAPI, os.Getenv("CSV_A"), "A"); err != nil {
		return finishImport(client, asyncWriteErrors, err)
	}
	if err := processCSV(writeAPI, os.Getenv("CSV_B"), "B"); err != nil {
		return finishImport(client, asyncWriteErrors, err)
	}
	if err := processCSV(writeAPI, os.Getenv("CSV_C"), "C"); err != nil {
		return finishImport(client, asyncWriteErrors, err)
	}

	if err := finishImport(client, asyncWriteErrors, nil); err != nil {
		return err
	}

	log.Print("Done!")
	return nil
}

func finishImport(client influxdb2.Client, asyncWriteErrors <-chan error, err error) error {
	client.Close()
	return errors.Join(err, <-asyncWriteErrors)
}

func collectAsyncWriteErrors(errCh <-chan error) <-chan error {
	done := make(chan error, 1)
	go func() {
		var errs []error
		for err := range errCh {
			if err != nil {
				errs = append(errs, err)
			}
		}
		done <- errors.Join(errs...)
	}()
	return done
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

func downloadCsv(url string) (string, error) {
	if url == "" {
		return "", fmt.Errorf("csv url is empty")
	}

	tempDir := os.TempDir()
	if err := os.MkdirAll(tempDir, 0o755); err != nil {
		return "", fmt.Errorf("create temp dir %s for %s: %w", tempDir, url, err)
	}

	out, err := os.CreateTemp(tempDir, "shelly-*.csv")
	if err != nil {
		return "", fmt.Errorf("create temp csv for %s: %w", url, err)
	}
	tempFile := out.Name()
	log.Printf("Downloading csv %s to %s\n", url, tempFile)

	resp, err := downloadClient.Get(url)
	if err != nil {
		_ = out.Close()
		_ = os.Remove(tempFile)
		return "", fmt.Errorf("download csv %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		_ = out.Close()
		_ = os.Remove(tempFile)
		return "", fmt.Errorf("download csv %s: unexpected HTTP status %s", url, resp.Status)
	}

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		_ = out.Close()
		_ = os.Remove(tempFile)
		return "", fmt.Errorf("copy csv %s: %w", url, err)
	}
	if err := out.Close(); err != nil {
		_ = os.Remove(tempFile)
		return "", fmt.Errorf("close temp csv for %s: %w", url, err)
	}

	log.Printf("Downloaded csv %s to %s\n", url, tempFile)

	return tempFile, nil
}

func processCSV(writeAPI api.WriteAPI, url, phase string) error {
	path, err := downloadCsv(url)
	if err != nil {
		return err
	}
	defer os.Remove(path)

	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open csv %s: %w", path, err)
	}
	defer f.Close()

	counter, err := processCSVReader(writeAPI, f, path, phase)
	if err != nil {
		return err
	}

	writeAPI.Flush()
	log.Printf("Points written for csv %s and phase %s: %d\n", path, phase, counter)
	return nil
}

func processCSVReader(writeAPI api.WriteAPI, r io.Reader, source, phase string) (int, error) {
	csvReader := csv.NewReader(r)
	counter := 0

	if header, err := csvReader.Read(); err != nil {
		return 0, fmt.Errorf("read csv header from %s: %w", source, err)
	} else {
		if len(header) > 0 && header[0] == "Another file transfer is in progress!" {
			return 0, fmt.Errorf("another file transfer is in progress: %s", source)
		}
	}

	for {
		rec, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return counter, fmt.Errorf("read csv record %d from %s: %w", counter+1, source, err)
		}
		if len(rec) < 3 {
			return counter, fmt.Errorf("read csv record %d from %s: expected at least 3 columns, got %d", counter+1, source, len(rec))
		}

		consumed, err := parseFloat(rec[1])
		if err != nil {
			return counter, fmt.Errorf("parse consumed value on record %d from %s: %w", counter+1, source, err)
		}
		returned, err := parseFloat(rec[2])
		if err != nil {
			return counter, fmt.Errorf("parse returned value on record %d from %s: %w", counter+1, source, err)
		}
		timestamp, err := parseTime(rec[0])
		if err != nil {
			return counter, fmt.Errorf("parse timestamp on record %d from %s: %w", counter+1, source, err)
		}

		ep := EnergyPoint{
			Phase:     phase,
			Consumed:  consumed,
			Returned:  returned,
			Timestamp: timestamp,
		}

		writePoint(writeAPI, ep)
		counter++
	}

	return counter, nil
}
