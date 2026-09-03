package geo

import (
	"testing"
)

func TestHaversine(t *testing.T) {
	// Distance between New York and London
	// NY: 40.7128° N, 74.0060° W
	// London: 51.5074° N, 0.1278° W
	// Approx distance: 5570 km

	lat1, lng1 := 40.7128, -74.0060
	lat2, lng2 := 51.5074, -0.1278

	dist := Haversine(lat1, lng1, lat2, lng2)

	if dist < 5500 || dist > 5600 {
		t.Errorf("Expected distance around 5570km, got %f", dist)
	}
}

func TestCalculateETA(t *testing.T) {
	// 60 km distance at 60 km/h should be 60 minutes
	// 1 degree lat is approx 111km.
	// Let's use simple points.

	lat1, lng1 := 0.0, 0.0
	lat2, lng2 := 1.0, 0.0 // Approx 111.19 km

	speed := 111.19
	eta := CalculateETA(lat1, lng1, lat2, lng2, speed)

	if eta < 59 || eta > 61 {
		t.Errorf("Expected ETA around 60 mins, got %d", eta)
	}

	// Test default speed
	etaDefault := CalculateETA(lat1, lng1, lat2, lng2, 0.5) // Speed < 1, should use 20 km/h
	// 111.19 km / 20 km/h = 5.55 hours = 333 mins
	if etaDefault < 330 || etaDefault > 340 {
		t.Errorf("Expected ETA around 333 mins for default speed, got %d", etaDefault)
	}
}
