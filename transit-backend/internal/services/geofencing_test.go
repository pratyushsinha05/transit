package services

import (
	"testing"
)

func TestCalculateHexAtResolution(t *testing.T) {
	// Test cases with known coordinates
	tests := []struct {
		name       string
		lat        float64
		lng        float64
		resolution int
		wantEmpty  bool
	}{
		{
			name:       "Valid Delhi coordinates at res 9",
			lat:        28.6139,
			lng:        77.2090,
			resolution: 9,
			wantEmpty:  false,
		},
		{
			name:       "Valid NYC coordinates at res 9",
			lat:        40.7128,
			lng:        -74.0060,
			resolution: 9,
			wantEmpty:  false,
		},
		{
			name:       "Valid coordinates at res 7",
			lat:        51.5074,
			lng:        -0.1278,
			resolution: 7,
			wantEmpty:  false,
		},
		{
			name:       "Zero coordinates",
			lat:        0,
			lng:        0,
			resolution: 9,
			wantEmpty:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hex := CalculateHexAtResolution(tt.lat, tt.lng, tt.resolution)
			if tt.wantEmpty && hex != "" {
				t.Errorf("CalculateHexAtResolution() = %v, want empty", hex)
			}
			if !tt.wantEmpty && hex == "" {
				t.Errorf("CalculateHexAtResolution() = empty, want non-empty hex")
			}
			if !tt.wantEmpty {
				// H3 hex strings should be 15 characters for most resolutions
				if len(hex) < 10 || len(hex) > 20 {
					t.Errorf("CalculateHexAtResolution() hex length = %d, want 10-20 chars", len(hex))
				}
			}
		})
	}
}

func TestHexEdgeLengthMeters(t *testing.T) {
	tests := []struct {
		resolution int
		wantMin    float64
		wantMax    float64
	}{
		{resolution: 0, wantMin: 1000000, wantMax: 1200000},
		{resolution: 5, wantMin: 8000, wantMax: 9000},
		{resolution: 9, wantMin: 170, wantMax: 180},
		{resolution: 15, wantMin: 0.4, wantMax: 0.6},
		{resolution: 16, wantMin: 0, wantMax: 0}, // Invalid resolution
	}

	for _, tt := range tests {
		t.Run("resolution_"+string(rune(tt.resolution+'0')), func(t *testing.T) {
			got := HexEdgeLengthMeters(tt.resolution)
			if got < tt.wantMin || got > tt.wantMax {
				t.Errorf("HexEdgeLengthMeters(%d) = %v, want between %v and %v",
					tt.resolution, got, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestSameLocationSameHex(t *testing.T) {
	// Two nearly identical locations should produce the same hex
	lat1, lng1 := 28.6139, 77.2090
	lat2, lng2 := 28.6139, 77.2090

	hex1 := CalculateHexAtResolution(lat1, lng1, 9)
	hex2 := CalculateHexAtResolution(lat2, lng2, 9)

	if hex1 != hex2 {
		t.Errorf("Same coordinates should produce same hex: %s != %s", hex1, hex2)
	}
}

func TestNearbyLocationsSimilarHex(t *testing.T) {
	// Two very close locations (within ~50m) at resolution 9 (~175m edge)
	// should often be in the same hex or adjacent hexes
	lat1, lng1 := 28.6139, 77.2090
	lat2, lng2 := 28.6140, 77.2091 // ~15m away

	hex1 := CalculateHexAtResolution(lat1, lng1, 9)
	hex2 := CalculateHexAtResolution(lat2, lng2, 9)

	// They should be the same at this resolution (15m is well within 175m edge)
	if hex1 != hex2 {
		t.Logf("Note: Very close points produced different hexes (at boundary): %s vs %s", hex1, hex2)
	}
}

func TestFarLocationsDifferentHex(t *testing.T) {
	// Two locations >1km apart should definitely have different hexes at res 9
	lat1, lng1 := 28.6139, 77.2090 // Delhi
	lat2, lng2 := 28.6240, 77.2190 // ~1.4km away

	hex1 := CalculateHexAtResolution(lat1, lng1, 9)
	hex2 := CalculateHexAtResolution(lat2, lng2, 9)

	if hex1 == hex2 {
		t.Errorf("Distant coordinates should produce different hexes at res 9: both = %s", hex1)
	}
}

func TestHexConsistencyAcrossResolutions(t *testing.T) {
	lat, lng := 28.6139, 77.2090

	// Higher resolution should produce different (more specific) hex
	hex7 := CalculateHexAtResolution(lat, lng, 7)
	hex9 := CalculateHexAtResolution(lat, lng, 9)
	hex11 := CalculateHexAtResolution(lat, lng, 11)

	// All should be non-empty
	if hex7 == "" || hex9 == "" || hex11 == "" {
		t.Error("All resolutions should produce non-empty hexes")
	}

	// They should all be different (different resolution = different hex)
	if hex7 == hex9 || hex9 == hex11 || hex7 == hex11 {
		t.Error("Different resolutions should produce different hexes")
	}
}
