package api

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/andrew/avweather_cache/cache"
	"github.com/andrew/avweather_cache/metrics"
	"github.com/andrew/avweather_cache/models"
	"gopkg.in/yaml.v3"
)

// Handler handles API requests
type Handler struct {
	cache *cache.Cache
}

// New creates a new API handler
func New(c *cache.Cache) *Handler {
	return &Handler{cache: c}
}

// MetarHandler handles GET /api/metar requests
func (h *Handler) MetarHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	defer func() {
		metrics.QueryLatency.Observe(time.Since(start).Seconds())
		metrics.TotalQueries.Inc()
	}()

	// Parse query parameters
	stationsParam := r.URL.Query().Get("stations")
	if stationsParam == "" {
		http.Error(w, "stations parameter is required", http.StatusBadRequest)
		return
	}

	stations := parseCommaSeparated(stationsParam)
	fieldsParam := r.URL.Query().Get("fields")
	var fields []string
	if fieldsParam != "" {
		fields = parseCommaSeparated(fieldsParam)
	}

	hoursBeforeNow := 0
	if hoursParam := r.URL.Query().Get("hoursBeforeNow"); hoursParam != "" {
		h, err := strconv.Atoi(hoursParam)
		if err != nil {
			http.Error(w, "invalid hoursBeforeNow parameter", http.StatusBadRequest)
			return
		}
		hoursBeforeNow = h
	}

	format := r.URL.Query().Get("format")
	if format == "" {
		format = "json"
	}
	if format != "json" && format != "csv" && format != "yaml" {
		http.Error(w, "invalid format parameter (must be json, csv, or yaml)", http.StatusBadRequest)
		return
	}

	// Get data from cache
	allMetars := h.cache.Get(stations)

	// Track metrics for each requested station
	for _, station := range stations {
		metrics.QueriesByStation.WithLabelValues(station).Inc()
	}

	// Filter by age if specified
	var filteredMetars []models.METAR
	cutoffTime := time.Now().Add(-time.Duration(hoursBeforeNow) * time.Hour)

	foundStations := make(map[string]bool)
	for _, metar := range allMetars {
		foundStations[metar.StationID] = true
		if hoursBeforeNow == 0 || metar.ObservationTime.After(cutoffTime) {
			filteredMetars = append(filteredMetars, metar)
		} else {
			metrics.StationsFilteredByAge.WithLabelValues(metar.StationID).Inc()
		}
	}

	// Track stations not in cache
	for _, station := range stations {
		if !foundStations[station] {
			metrics.StationsNotCached.WithLabelValues(station).Inc()
		}
	}

	// Filter fields if specified
	if len(fields) > 0 {
		filteredMetars = filterFields(filteredMetars, fields)
	}

	// Return in requested format
	switch format {
	case "json":
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(filteredMetars)
	case "yaml":
		w.Header().Set("Content-Type", "application/x-yaml")
		yaml.NewEncoder(w).Encode(filteredMetars)
	case "csv":
		w.Header().Set("Content-Type", "text/csv")
		if err := writeCSV(w, filteredMetars, fields); err != nil {
			http.Error(w, fmt.Sprintf("failed to write CSV: %v", err), http.StatusInternalServerError)
		}
	}
}

// parseCommaSeparated splits a comma-separated string into a slice
func parseCommaSeparated(s string) []string {
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// filterFields returns METARs with only the specified fields populated
func filterFields(metars []models.METAR, fields []string) []models.METAR {
	// Build a map of requested fields for quick lookup
	fieldMap := make(map[string]bool)
	for _, f := range fields {
		fieldMap[strings.ToLower(f)] = true
	}

	// Always include station_id as it's the key
	fieldMap["station_id"] = true

	result := make([]models.METAR, len(metars))
	for i, metar := range metars {
		filtered := models.METAR{StationID: metar.StationID}

		// Use reflection to copy only requested fields
		v := reflect.ValueOf(metar)
		fv := reflect.ValueOf(&filtered).Elem()
		t := v.Type()

		for j := 0; j < v.NumField(); j++ {
			field := t.Field(j)
			jsonTag := field.Tag.Get("json")
			if jsonTag == "" {
				continue
			}
			// Extract field name from json tag
			fieldName := strings.Split(jsonTag, ",")[0]
			if fieldMap[strings.ToLower(fieldName)] {
				fv.Field(j).Set(v.Field(j))
			}
		}

		result[i] = filtered
	}

	return result
}

// writeCSV writes METARs to CSV format
func writeCSV(w http.ResponseWriter, metars []models.METAR, fields []string) error {
	writer := csv.NewWriter(w)
	defer writer.Flush()

	if len(metars) == 0 {
		return nil
	}

	// Determine fields to write
	var headers []string
	if len(fields) > 0 {
		headers = fields
	} else {
		// Use all fields from the first METAR
		headers = getAllFieldNames()
	}

	// Write header
	if err := writer.Write(headers); err != nil {
		return err
	}

	// Write data rows
	for _, metar := range metars {
		row := make([]string, len(headers))
		for i, header := range headers {
			row[i] = getFieldValue(metar, header)
		}
		if err := writer.Write(row); err != nil {
			return err
		}
	}

	return nil
}

// getAllFieldNames returns all field names from METAR struct
func getAllFieldNames() []string {
	return []string{
		"station_id", "raw_text", "observation_time", "latitude", "longitude",
		"temp_c", "dewpoint_c", "wind_dir_degrees", "wind_speed_kt", "wind_gust_kt",
		"visibility_statute_mi", "altim_in_hg", "wx_string", "flight_category",
		"metar_type", "elevation_m", "precip_in",
	}
}

// getFieldValue retrieves a field value from a METAR as a string
func getFieldValue(metar models.METAR, fieldName string) string {
	switch strings.ToLower(fieldName) {
	case "station_id":
		return metar.StationID
	case "raw_text":
		return metar.RawText
	case "observation_time":
		if !metar.ObservationTime.IsZero() {
			return metar.ObservationTime.Format(time.RFC3339)
		}
		return ""
	case "latitude":
		return fmt.Sprintf("%.4f", metar.Latitude)
	case "longitude":
		return fmt.Sprintf("%.4f", metar.Longitude)
	case "temp_c":
		if metar.TempC != nil {
			return fmt.Sprintf("%.1f", *metar.TempC)
		}
		return ""
	case "dewpoint_c":
		if metar.DewpointC != nil {
			return fmt.Sprintf("%.1f", *metar.DewpointC)
		}
		return ""
	case "wind_dir_degrees":
		return metar.WindDirDegrees
	case "wind_speed_kt":
		if metar.WindSpeedKt != nil {
			return fmt.Sprintf("%d", *metar.WindSpeedKt)
		}
		return ""
	case "wind_gust_kt":
		if metar.WindGustKt != nil {
			return fmt.Sprintf("%d", *metar.WindGustKt)
		}
		return ""
	case "visibility_statute_mi":
		return metar.VisibilityMi
	case "altim_in_hg":
		if metar.AltimeterInHg != nil {
			return fmt.Sprintf("%.2f", *metar.AltimeterInHg)
		}
		return ""
	case "wx_string":
		return metar.WxString
	case "flight_category":
		return metar.FlightCategory
	case "metar_type":
		return metar.MetarType
	case "elevation_m":
		if metar.ElevationM != nil {
			return fmt.Sprintf("%.0f", *metar.ElevationM)
		}
		return ""
	case "precip_in":
		if metar.PrecipIn != nil {
			return fmt.Sprintf("%.2f", *metar.PrecipIn)
		}
		return ""
	default:
		return ""
	}
}
