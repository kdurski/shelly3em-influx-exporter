# Use an official Golang runtime as a parent image
FROM golang:1.20 AS builder

RUN apt-get update && apt-get upgrade -y

# Set the working directory to /app
WORKDIR /app

# Copy the current directory contents into the container at /app
COPY . /app

# Build the app
RUN go clean --modcache \
    && go mod tidy \
    && go build -o shelly cmd/shelly/*.go

FROM debian:12-slim

# Copy the binary from the builder stage into the container
COPY --from=builder /app/shelly /usr/local/bin/shelly
COPY --from=builder /app/.env .

ENTRYPOINT ["/usr/local/bin/shelly"]
