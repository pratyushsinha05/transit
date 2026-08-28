package hub

import (
	"encoding/json"
	"testing"
)

// TestMessage_JSONShape pins the canonical envelope from CLAUDE.md Sec 7.2:
// flat, no `data` wrapper, and -- the regression this test exists to catch
// (DEFECT-5) -- no `omitempty` on numeric fields, so a stopped device still
// serializes "speed": 0 instead of dropping the key.
func TestMessage_JSONShape(t *testing.T) {
	msg := Message{
		Type:      MsgTypeLocationUpdate,
		DeviceID:  "bus-001",
		RouteID:   "route-1",
		Latitude:  28.6139,
		Longitude: 77.2090,
		Speed:     0,
		Accuracy:  5.0,
		H3Hex:     "893da11408fffff",
		Timestamp: 1700000000,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var got map[string]interface{}
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	wantKeys := []string{
		"type", "device_id", "route_id", "latitude", "longitude",
		"speed", "accuracy", "h3_hex", "timestamp",
	}
	if len(got) != len(wantKeys) {
		t.Fatalf("got %d top-level keys, want %d (no data wrapper, no extra fields): %v", len(got), len(wantKeys), got)
	}
	for _, k := range wantKeys {
		if _, ok := got[k]; !ok {
			t.Errorf("missing key %q in serialized message: %v", k, got)
		}
	}

	if _, hasDataWrapper := got["data"]; hasDataWrapper {
		t.Error("message has a \"data\" wrapper key -- envelope must be flat per CLAUDE.md Sec 7.2")
	}
	if _, hasHeading := got["heading"]; hasHeading {
		t.Error("message has a \"heading\" key -- heading does not exist anywhere in this codebase, must not be sent")
	}

	speed, ok := got["speed"]
	if !ok {
		t.Fatal("speed key missing entirely -- omitempty is back (DEFECT-5 regression)")
	}
	if speed != float64(0) {
		t.Errorf("speed = %v, want 0", speed)
	}
}

func TestMessage_ZeroCoordinatesNotOmitted(t *testing.T) {
	// Equator/prime-meridian coordinates (0, 0) are the other omitempty
	// hazard: legitimate data that must not vanish from the payload.
	msg := Message{Type: MsgTypeLocationUpdate, DeviceID: "d1", Latitude: 0, Longitude: 0}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var got map[string]interface{}
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	for _, k := range []string{"latitude", "longitude", "accuracy", "timestamp"} {
		if _, ok := got[k]; !ok {
			t.Errorf("zero-valued field %q was omitted from the payload", k)
		}
	}
}
