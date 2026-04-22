package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/url"
	"os"
	"time"
)

const connectivityCheckTimeout = 5 * time.Second

var tcpDialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
	var dialer net.Dialer
	return dialer.DialContext(ctx, network, address)
}

type connectivityTarget struct {
	name string
	url  string
}

func checkConfiguredConnectivity() error {
	targets := []connectivityTarget{
		{name: "InfluxDB", url: os.Getenv("INFLUXDB_URL")},
		{name: "CSV A", url: os.Getenv("CSV_A")},
		{name: "CSV B", url: os.Getenv("CSV_B")},
		{name: "CSV C", url: os.Getenv("CSV_C")},
	}

	checked := make(map[string]string, len(targets))
	for _, target := range targets {
		address, err := connectivityAddress(target.url)
		if err != nil {
			return fmt.Errorf("%s connectivity check: %w", target.name, err)
		}
		if originalTarget, ok := checked[address]; ok {
			log.Printf("Connectivity already verified for %s via %s; skipping duplicate check for %s", address, originalTarget, target.name)
			continue
		}
		log.Printf("Checking TCP connectivity for %s at %s", target.name, address)
		if err := checkTCPConnectivity(address); err != nil {
			return fmt.Errorf("%s connectivity check: %w", target.name, err)
		}
		checked[address] = target.name
	}

	log.Print("Connectivity checks passed.")
	return nil
}

func connectivityAddress(rawURL string) (string, error) {
	if rawURL == "" {
		return "", fmt.Errorf("url is empty")
	}

	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("parse url %q: %w", rawURL, err)
	}
	if parsedURL.Host == "" {
		return "", fmt.Errorf("url %q does not include a host", rawURL)
	}

	port := parsedURL.Port()
	switch parsedURL.Scheme {
	case "http":
		if port == "" {
			port = "80"
		}
	case "https":
		if port == "" {
			port = "443"
		}
	default:
		return "", fmt.Errorf("url %q uses unsupported scheme %q", rawURL, parsedURL.Scheme)
	}

	host := parsedURL.Hostname()
	if host == "" {
		return "", fmt.Errorf("url %q does not include a hostname", rawURL)
	}

	return net.JoinHostPort(host, port), nil
}

func checkTCPConnectivity(address string) error {
	ctx, cancel := context.WithTimeout(context.Background(), connectivityCheckTimeout)
	defer cancel()

	conn, err := tcpDialContext(ctx, "tcp", address)
	if err != nil {
		return fmt.Errorf("dial tcp %s: %w", address, err)
	}
	if err := conn.Close(); err != nil {
		return fmt.Errorf("close tcp connection %s: %w", address, err)
	}
	return nil
}
