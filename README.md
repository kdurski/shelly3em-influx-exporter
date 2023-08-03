# shelly3em-influx-exporter

Export Shelly 3EM data to InfluxDB

1. Create `.env.local`
2. Execute `go run cmd/shelly/*.go --dry-run` to see if env variables are picked up
3. Execute `go run cmd/shelly/*.go` to run the exporter

Alternatively

1. Create `.env.local`
2. Build image via `docker buildx build -f Dockerfile . -t shelly:latest`
3. Execute `docker run --rm --env-file=.env.local shelly:latest --dry-run` to see if env variables are picked up
4. Execute `docker run --rm --env-file=.env.local shelly:latest` to run the exporter

Do not the app in parallel! Shelly3EM can only handle 1 CSV download at a time. 