package main

import (
	"context"
	"errors"
	"net"
	"strings"
	"testing"
	"time"
)

type fakeConn struct{}

func (fakeConn) Read(_ []byte) (int, error)         { return 0, nil }
func (fakeConn) Write(_ []byte) (int, error)        { return 0, nil }
func (fakeConn) Close() error                       { return nil }
func (fakeConn) LocalAddr() net.Addr                { return &net.TCPAddr{} }
func (fakeConn) RemoteAddr() net.Addr               { return &net.TCPAddr{} }
func (fakeConn) SetDeadline(_ time.Time) error      { return nil }
func (fakeConn) SetReadDeadline(_ time.Time) error  { return nil }
func (fakeConn) SetWriteDeadline(_ time.Time) error { return nil }

func TestConnectivityAddressDefaultsPorts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		rawURL string
		want   string
	}{
		{name: "http", rawURL: "http://192.168.2.10/emeter/0/emeter.csv", want: "192.168.2.10:80"},
		{name: "https", rawURL: "https://influx.example.org/api/v2/write", want: "influx.example.org:443"},
		{name: "custom port", rawURL: "http://influx.example.org:8086", want: "influx.example.org:8086"},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := connectivityAddress(test.rawURL)
			if err != nil {
				t.Fatalf("connectivityAddress(%q) error = %v", test.rawURL, err)
			}
			if got != test.want {
				t.Fatalf("connectivityAddress(%q) = %q, want %q", test.rawURL, got, test.want)
			}
		})
	}
}

func TestConnectivityAddressRejectsUnsupportedScheme(t *testing.T) {
	t.Parallel()

	_, err := connectivityAddress("ftp://example.org/export.csv")
	if err == nil {
		t.Fatal("connectivityAddress error = nil, want error")
	}
}

func TestCheckConfiguredConnectivityDialsUniqueTargets(t *testing.T) {
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
		if network != "tcp" {
			t.Fatalf("network = %q, want tcp", network)
		}
		dialed = append(dialed, address)
		return fakeConn{}, nil
	}

	if err := checkConfiguredConnectivity(); err != nil {
		t.Fatalf("checkConfiguredConnectivity error = %v", err)
	}

	if len(dialed) != 2 {
		t.Fatalf("dialed addresses = %v, want 2 unique targets", dialed)
	}
	if dialed[0] != "influx.example.org:8086" {
		t.Fatalf("first dialed address = %q, want influx.example.org:8086", dialed[0])
	}
	if dialed[1] != "192.168.2.10:80" {
		t.Fatalf("second dialed address = %q, want 192.168.2.10:80", dialed[1])
	}
}

func TestCheckConfiguredConnectivityReturnsDialError(t *testing.T) {
	t.Setenv("INFLUXDB_URL", "http://influx.example.org:8086")
	t.Setenv("CSV_A", "http://192.168.2.10/emeter/0/emeter.csv")
	t.Setenv("CSV_B", "http://192.168.2.10/emeter/1/emeter.csv")
	t.Setenv("CSV_C", "http://192.168.2.10/emeter/2/emeter.csv")

	originalDialer := tcpDialContext
	t.Cleanup(func() {
		tcpDialContext = originalDialer
	})

	tcpDialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		if address == "192.168.2.10:80" {
			return nil, errors.New("no route to host")
		}
		return fakeConn{}, nil
	}

	err := checkConfiguredConnectivity()
	if err == nil {
		t.Fatal("checkConfiguredConnectivity error = nil, want error")
	}
	if got := err.Error(); got == "" || !containsAll(got, "CSV A", "192.168.2.10:80", "no route to host") {
		t.Fatalf("error = %q, want target name, address, and dial failure", got)
	}
}

func containsAll(value string, parts ...string) bool {
	for _, part := range parts {
		if !strings.Contains(value, part) {
			return false
		}
	}
	return true
}
