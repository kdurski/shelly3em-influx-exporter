# shelly3em-influx-exporter

Export Shelly 3EM data to InfluxDB.

This small Go application connects to your Shelly 3EM device, downloads energy data export CSV files for each of the 3 phases, and writes the energy metrics (consumed and returned) into an InfluxDB instance. 

### Configuration

The application is configured using environment variables. These are typically set in a `.env.local` file for development and native execution. It requires the following variables:

**InfluxDB Connection:**
- `INFLUXDB_URL`: The URL of your InfluxDB instance.
- `INFLUXDB_TOKEN`: API token with write access.
- `INFLUXDB_ORG`: Your InfluxDB organization name.
- `INFLUXDB_BUCKET`: The bucket where data will be written.

**Shelly 3EM Endpoints:**
The endpoints to download the CSV for each phase. Typically in the format `http://<shelly-ip>/emeter/0/emeter.csv`, `.../1/...`, `.../2/...`.
- `CSV_A`: URL for phase A.
- `CSV_B`: URL for phase B.
- `CSV_C`: URL for phase C.

### Usage

**Native Execution (Go):**
1. Ensure Go is installed (version 1.20+).
2. Create `.env.local` and configure your variables.
3. Execute `go run cmd/shelly/*.go --help` to see the available flags.
4. Execute `go run cmd/shelly/*.go --dry-run` to see if env variables are picked up without writing to the database.
5. Execute `go run cmd/shelly/*.go --check-connectivity` to verify TCP reachability to InfluxDB and the Shelly host without starting CSV generation.
6. Execute `go run cmd/shelly/*.go` to run the exporter.

**Docker:**
Alternatively, you can build and run the application via Docker:

1. Create `.env.local` with your configuration.
2. Build the image: `docker buildx build -f Dockerfile . -t shelly:latest`
3. Show CLI help: `docker run --rm shelly:latest --help`
4. Test configuration: `docker run --rm --env-file=.env.local shelly:latest --dry-run`
5. Verify connectivity: `docker run --rm --env-file=.env.local shelly:latest --check-connectivity`
6. Run the exporter: `docker run --rm --env-file=.env.local shelly:latest`

**Note:** Do not run multiple instances of the application in parallel! Shelly3EM can only handle 1 CSV download at a time.

Before starting any CSV download, the exporter performs a TCP connectivity preflight against InfluxDB and the configured Shelly host(s). This verifies that the target hosts are reachable without calling the CSV endpoints themselves.
