package main

import (
	"context"
	"net"
	"testing"
)

func TestRunCheckConnectivityMode(t *testing.T) {
	t.Setenv("INFLUXDB_URL", "http://influx.example.org:8086")
	t.Setenv("CSV_A", "http://192.168.2.10/emeter/0/emeter.csv")
	t.Setenv("CSV_B", "http://192.168.2.10/emeter/1/emeter.csv")
	t.Setenv("CSV_C", "http://192.168.2.10/emeter/2/emeter.csv")

	originalDialer := tcpDialContext
	t.Cleanup(func() {
		tcpDialContext = originalDialer
	})

	var dialed []string
	tcpDialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		dialed = append(dialed, address)
		return fakeConn{}, nil
	}

	if err := run(false, true); err != nil {
		t.Fatalf("run(false, true) error = %v", err)
	}
	if len(dialed) != 2 {
		t.Fatalf("dialed addresses = %v, want 2 unique targets", dialed)
	}
}
