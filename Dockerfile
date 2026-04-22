# Use an official Golang runtime as a parent image
FROM golang:1.25-alpine AS builder

# Install CA certificates
RUN apk --no-cache add ca-certificates && update-ca-certificates

# Set the working directory to /app
WORKDIR /app

# Copy the current directory contents into the container at /app
COPY . .

# Build the static binary
RUN go clean --modcache \
    && go mod tidy \
    && CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o shelly cmd/shelly/*.go

# Use scratch for the smallest possible final image
FROM scratch

# Copy CA certificates so HTTPS requests work
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy the binary from the builder stage into the container
COPY --from=builder /app/shelly /usr/local/bin/shelly

# Copy the default configuration
COPY --from=builder /app/.env /.env

ENTRYPOINT ["/usr/local/bin/shelly"]
