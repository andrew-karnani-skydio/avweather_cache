package cache

import (
	"compress/gzip"
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/andrew/avweather_cache/models"
)

func TestCacheUpdate(t *testing.T) {
	// Read test data
	data, err := os.ReadFile("../testdata/metars.cache.xml")
	if err != nil {
		t.Fatalf("Failed to read test data: %v", err)
	}

	// Create test server that serves gzipped XML
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-gzip")
		gzWriter := gzip.NewWriter(w)
		defer gzWriter.Close()
		gzWriter.Write(data)
	}))
	defer server.Close()

	// Create cache
	c := New(server.URL, 1*time.Minute)

	// Do initial update
	if err := c.update(); err != nil {
		t.Fatalf("Initial update failed: %v", err)
	}

	// Verify cache has data
	if len(c.data) == 0 {
		t.Fatal("Cache is empty after update")
	}

	initialCount := len(c.data)
	t.Logf("Initial cache size: %d", initialCount)

	// Verify we can get specific stations
	testStations := []string{"KBEA", "KRCM", "TXKF"}
	metars := c.Get(testStations)
	if len(metars) != len(testStations) {
		t.Errorf("Expected %d METARs, got %d", len(testStations), len(metars))
	}

	// Verify data integrity
	for _, metar := range metars {
		if metar.StationID == "" {
			t.Error("Empty station ID")
		}
		if metar.ObservationTime.IsZero() {
			t.Error("Zero observation time")
		}
	}
}

func TestCacheMergeNotPurge(t *testing.T) {
	// Create test data with two stations
	data1 := `<?xml version="1.0"?><response><data>
		<METAR>
			<station_id>KTEST1</station_id>
			<observation_time>2026-01-26T12:00:00Z</observation_time>
			<raw_text>METAR KTEST1 261200Z</raw_text>
		</METAR>
	</data></response>`

	data2 := `<?xml version="1.0"?><response><data>
		<METAR>
			<station_id>KTEST2</station_id>
			<observation_time>2026-01-26T12:05:00Z</observation_time>
			<raw_text>METAR KTEST2 261205Z</raw_text>
		</METAR>
	</data></response>`

	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-gzip")
		gzWriter := gzip.NewWriter(w)
		defer gzWriter.Close()

		if callCount == 0 {
			gzWriter.Write([]byte(data1))
		} else {
			gzWriter.Write([]byte(data2))
		}
		callCount++
	}))
	defer server.Close()

	c := New(server.URL, 1*time.Minute)

	// First update
	if err := c.update(); err != nil {
		t.Fatalf("First update failed: %v", err)
	}

	if len(c.data) != 1 {
		t.Fatalf("Expected 1 station after first update, got %d", len(c.data))
	}
	if _, ok := c.data["KTEST1"]; !ok {
		t.Error("KTEST1 not in cache")
	}

	// Second update - should merge, not replace
	if err := c.update(); err != nil {
		t.Fatalf("Second update failed: %v", err)
	}

	if len(c.data) != 2 {
		t.Fatalf("Expected 2 stations after second update (merge), got %d", len(c.data))
	}
	if _, ok := c.data["KTEST1"]; !ok {
		t.Error("KTEST1 not in cache after second update (should have been preserved)")
	}
	if _, ok := c.data["KTEST2"]; !ok {
		t.Error("KTEST2 not in cache after second update")
	}
}

func TestCacheMetrics(t *testing.T) {
	// Create cache with mock data
	c := New("http://example.com", 1*time.Minute)

	now := time.Now()
	c.data = map[string]models.METAR{
		"KTEST1": {
			StationID:       "KTEST1",
			ObservationTime: now.Add(-30 * time.Minute),
		},
		"KTEST2": {
			StationID:       "KTEST2",
			ObservationTime: now.Add(-90 * time.Minute),
		},
		"KTEST3": {
			StationID:       "KTEST3",
			ObservationTime: now.Add(-3 * time.Hour),
		},
	}

	// Update metrics
	c.updateMetrics()

	// Verify metrics were calculated (we can't easily test the actual prometheus values,
	// but we can verify the method runs without error)
	status := c.Status()
	if status.TotalStations != 3 {
		t.Errorf("Expected 3 total stations, got %d", status.TotalStations)
	}
}

func TestParseRealData(t *testing.T) {
	// This test verifies we can parse the real XML structure
	data, err := os.ReadFile("../testdata/metars.cache.xml")
	if err != nil {
		t.Skipf("Skipping real data test: %v", err)
		return
	}

	var response models.MetarResponse
	if err := xml.Unmarshal(data, &response); err != nil {
		t.Fatalf("Failed to parse real XML data: %v", err)
	}

	if len(response.Data) == 0 {
		t.Fatal("No METARs parsed from real data")
	}

	t.Logf("Successfully parsed %d METARs from real data", len(response.Data))

	// Verify structure of first METAR
	metar := response.Data[0]
	if metar.StationID == "" {
		t.Error("First METAR has empty station ID")
	}
	if metar.ObservationTime.IsZero() {
		t.Error("First METAR has zero observation time")
	}
	t.Logf("Sample METAR: %s at %s", metar.StationID, metar.ObservationTime)
}
