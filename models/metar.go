package models

import (
	"time"
)

// MetarResponse represents the root XML response from aviationweather.gov
type MetarResponse struct {
	RequestIndex int     `xml:"request_index"`
	Data         []METAR `xml:"data>METAR"`
}

// METAR represents a single METAR observation
type METAR struct {
	RawText         string         `xml:"raw_text" json:"raw_text,omitempty" yaml:"raw_text,omitempty"`
	StationID       string         `xml:"station_id" json:"station_id,omitempty" yaml:"station_id,omitempty"`
	ObservationTime time.Time      `xml:"observation_time" json:"observation_time,omitempty" yaml:"observation_time,omitempty"`
	Latitude        float64        `xml:"latitude" json:"latitude,omitempty" yaml:"latitude,omitempty"`
	Longitude       float64        `xml:"longitude" json:"longitude,omitempty" yaml:"longitude,omitempty"`
	TempC           *float64       `xml:"temp_c" json:"temp_c,omitempty" yaml:"temp_c,omitempty"`
	DewpointC       *float64       `xml:"dewpoint_c" json:"dewpoint_c,omitempty" yaml:"dewpoint_c,omitempty"`
	WindDirDegrees  string         `xml:"wind_dir_degrees" json:"wind_dir_degrees,omitempty" yaml:"wind_dir_degrees,omitempty"`
	WindSpeedKt     *int           `xml:"wind_speed_kt" json:"wind_speed_kt,omitempty" yaml:"wind_speed_kt,omitempty"`
	WindGustKt      *int           `xml:"wind_gust_kt" json:"wind_gust_kt,omitempty" yaml:"wind_gust_kt,omitempty"`
	VisibilityMi    string         `xml:"visibility_statute_mi" json:"visibility_statute_mi,omitempty" yaml:"visibility_statute_mi,omitempty"`
	AltimeterInHg   *float64       `xml:"altim_in_hg" json:"altim_in_hg,omitempty" yaml:"altim_in_hg,omitempty"`
	WxString        string         `xml:"wx_string" json:"wx_string,omitempty" yaml:"wx_string,omitempty"`
	SkyConditions   []SkyCondition `xml:"sky_condition" json:"sky_conditions,omitempty" yaml:"sky_conditions,omitempty"`
	FlightCategory  string         `xml:"flight_category" json:"flight_category,omitempty" yaml:"flight_category,omitempty"`
	MetarType       string         `xml:"metar_type" json:"metar_type,omitempty" yaml:"metar_type,omitempty"`
	ElevationM      *float64       `xml:"elevation_m" json:"elevation_m,omitempty" yaml:"elevation_m,omitempty"`
	PrecipIn        *float64       `xml:"precip_in" json:"precip_in,omitempty" yaml:"precip_in,omitempty"`
}

// SkyCondition represents cloud layer information
type SkyCondition struct {
	SkyCover       string `xml:"sky_cover,attr" json:"sky_cover" yaml:"sky_cover"`
	CloudBaseFtAGL *int   `xml:"cloud_base_ft_agl,attr" json:"cloud_base_ft_agl,omitempty" yaml:"cloud_base_ft_agl,omitempty"`
}
