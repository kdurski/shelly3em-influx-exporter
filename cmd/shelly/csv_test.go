package main

import (
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/influxdata/influxdb-client-go/v2/api/write"
)

type fakeWriteAPI struct {
	points  []*write.Point
	flushed bool
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func (f *fakeWriteAPI) WriteRecord(line string) {}

func (f *fakeWriteAPI) WritePoint(point *write.Point) {
	f.points = append(f.points, point)
}

func (f *fakeWriteAPI) Flush() {
	f.flushed = true
}

func (f *fakeWriteAPI) Errors() <-chan error {
	errCh := make(chan error)
	close(errCh)
	return errCh
}

func (f *fakeWriteAPI) SetWriteFailedCallback(cb api.WriteFailedCallback) {
}

func TestProcessCSVReaderWritesPoints(t *testing.T) {
	t.Parallel()

	writeAPI := &fakeWriteAPI{}
	input := strings.NewReader("timestamp,consumed,returned\n2024-01-02 03:04,1.5,0.5\n2024-01-02 03:05,2.5,0.0\n")

	count, err := processCSVReader(writeAPI, input, "phase-a.csv", "A")
	if err != nil {
		t.Fatalf("processCSVReader returned error: %v", err)
	}
	if count != 2 {
		t.Fatalf("processCSVReader count = %d, want 2", count)
	}
	if len(writeAPI.points) != 2 {
		t.Fatalf("written points = %d, want 2", len(writeAPI.points))
	}

	first := writeAPI.points[0]
	if first.Name() != "energy" {
		t.Fatalf("measurement = %q, want energy", first.Name())
	}
	if got := first.TagList()[0].Value; got != "A" {
		t.Fatalf("phase tag = %q, want A", got)
	}
	if got := first.Time(); !got.Equal(time.Date(2024, time.January, 2, 3, 4, 0, 0, time.UTC)) {
		t.Fatalf("timestamp = %s, want 2024-01-02 03:04:00 +0000 UTC", got)
	}
	if got := first.FieldList()[0].Value; got != float64(1.5) {
		t.Fatalf("consumed field = %v, want 1.5", got)
	}
	if got := first.FieldList()[1].Value; got != float64(0.5) {
		t.Fatalf("returned field = %v, want 0.5", got)
	}
}

func TestProcessCSVReaderRejectsShellyBusyResponse(t *testing.T) {
	t.Parallel()

	writeAPI := &fakeWriteAPI{}
	input := strings.NewReader("Another file transfer is in progress!\n")

	_, err := processCSVReader(writeAPI, input, "busy.csv", "A")
	if err == nil {
		t.Fatal("processCSVReader error = nil, want error")
	}
	if !strings.Contains(err.Error(), "another file transfer is in progress") {
		t.Fatalf("error = %q, want transfer-in-progress message", err.Error())
	}
}

func TestDownloadCSVRejectsNonOKStatus(t *testing.T) {
	t.Parallel()

	originalClient := downloadClient
	downloadClient = &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusServiceUnavailable,
				Status:     "503 Service Unavailable",
				Body:       io.NopCloser(strings.NewReader("nope")),
				Header:     make(http.Header),
			}, nil
		}),
	}
	t.Cleanup(func() {
		downloadClient = originalClient
	})

	_, err := downloadCsv("http://example.invalid/emeter.csv")
	if err == nil {
		t.Fatal("downloadCsv error = nil, want error")
	}
	if !strings.Contains(err.Error(), "503 Service Unavailable") {
		t.Fatalf("error = %q, want HTTP status", err.Error())
	}
}

func TestDownloadCSVCreatesMissingTempDir(t *testing.T) {
	tmpRoot := t.TempDir()
	missingTempDir := filepath.Join(tmpRoot, "missing", "tmp")
	t.Setenv("TMPDIR", missingTempDir)

	originalClient := downloadClient
	downloadClient = &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Status:     "200 OK",
				Body:       io.NopCloser(strings.NewReader("timestamp,consumed,returned\n")),
				Header:     make(http.Header),
			}, nil
		}),
	}
	t.Cleanup(func() {
		downloadClient = originalClient
	})

	path, err := downloadCsv("http://example.invalid/emeter.csv")
	if err != nil {
		t.Fatalf("downloadCsv error = %v", err)
	}
	defer os.Remove(path)

	if !strings.HasPrefix(path, missingTempDir+string(os.PathSeparator)) {
		t.Fatalf("temp path = %q, want it under %q", path, missingTempDir)
	}
}

func TestCollectAsyncWriteErrorsAggregatesErrors(t *testing.T) {
	t.Parallel()

	errCh := make(chan error, 2)
	errCh <- errors.New("first")
	errCh <- errors.New("second")
	close(errCh)

	err := <-collectAsyncWriteErrors(errCh)
	if err == nil {
		t.Fatal("collectAsyncWriteErrors error = nil, want error")
	}
	if !strings.Contains(err.Error(), "first") || !strings.Contains(err.Error(), "second") {
		t.Fatalf("error = %q, want both async errors", err.Error())
	}
}
