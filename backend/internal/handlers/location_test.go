package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"transit-backend/internal/models"

	"github.com/labstack/echo/v4"
)

// mockIngestService is a test double for IngestService, shared by unit tests
// and benchmarks.
type mockIngestService struct {
	ingestFn  func(ctx context.Context, loc *models.Location) error
	callCount int
	lastLoc   *models.Location
}

func (m *mockIngestService) IngestLocation(ctx context.Context, loc *models.Location) error {
	m.callCount++
	m.lastLoc = loc
	if m.ingestFn != nil {
		return m.ingestFn(ctx, loc)
	}
	return nil
}

// newTestEchoContext creates an Echo context and response recorder for testing.
func newTestEchoContext(method, target string, body io.Reader) (echo.Context, *httptest.ResponseRecorder) {
	e := echo.New()
	req := httptest.NewRequest(method, target, body)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	return e.NewContext(req, rec), rec
}

func TestIngestLocation_HappyPath(t *testing.T) {
	mockSvc := &mockIngestService{
		ingestFn: func(ctx context.Context, loc *models.Location) error {
			loc.HexRes9 = "891f5a44a4bffff" // simulated side effect
			return nil
		},
	}
	handler := NewLocationHandler(mockSvc)

	payload := `{
		"device_id": "bus-101",
		"latitude": 28.6139,
		"longitude": 77.2090,
		"speed": 35.0,
		"accuracy": 4.5,
		"timestamp": 1672531200
	}`

	c, rec := newTestEchoContext(http.MethodPost, "/api/location", strings.NewReader(payload))
	err := handler.IngestLocation(c)
	if err != nil {
		t.Fatalf("IngestLocation returned unexpected error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp["status"] != "ok" {
		t.Errorf("status = %v, want ok", resp["status"])
	}
	if resp["hex_res9"] != "891f5a44a4bffff" {
		t.Errorf("hex_res9 = %v, want 891f5a44a4bffff", resp["hex_res9"])
	}

	if mockSvc.callCount != 1 {
		t.Errorf("service called %d times, want 1", mockSvc.callCount)
	}
	if mockSvc.lastLoc == nil || mockSvc.lastLoc.DeviceID != "bus-101" {
		t.Errorf("unexpected lastLoc: %+v", mockSvc.lastLoc)
	}
	if mockSvc.lastLoc.Latitude != 28.6139 || mockSvc.lastLoc.Longitude != 77.2090 {
		t.Errorf("coordinates mismatch: lat=%f, lng=%f", mockSvc.lastLoc.Latitude, mockSvc.lastLoc.Longitude)
	}
}

func TestIngestLocation_MissingDeviceID(t *testing.T) {
	mockSvc := &mockIngestService{}
	handler := NewLocationHandler(mockSvc)

	payload := `{"latitude": 28.6139, "longitude": 77.2090, "speed": 20.0}`
	c, rec := newTestEchoContext(http.MethodPost, "/api/location", strings.NewReader(payload))

	err := handler.IngestLocation(c)
	if err != nil {
		t.Fatalf("unexpected handler error: %v", err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	if !strings.Contains(rec.Body.String(), "device_id is required") {
		t.Errorf("body = %s, want device_id is required", rec.Body.String())
	}
	if mockSvc.callCount != 0 {
		t.Errorf("service called %d times, want 0", mockSvc.callCount)
	}
}

func TestIngestLocation_InvalidCoordinates(t *testing.T) {
	mockSvc := &mockIngestService{}
	handler := NewLocationHandler(mockSvc)

	tests := []struct {
		name string
		lat  float64
		lng  float64
	}{
		{"LatTooLow", -90.1, 77.2},
		{"LatTooHigh", 90.1, 77.2},
		{"LngTooLow", 28.6, -180.1},
		{"LngTooHigh", 28.6, 180.1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(models.Location{
				DeviceID:  "dev-1",
				Latitude:  tt.lat,
				Longitude: tt.lng,
				Speed:     10.0,
			})
			c, rec := newTestEchoContext(http.MethodPost, "/api/location", bytes.NewReader(body))
			err := handler.IngestLocation(c)
			if err != nil {
				t.Fatalf("unexpected handler error: %v", err)
			}
			if rec.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want %d for %s", rec.Code, http.StatusBadRequest, tt.name)
			}
			if !strings.Contains(rec.Body.String(), "invalid coordinates") {
				t.Errorf("body = %s, want invalid coordinates", rec.Body.String())
			}
		})
	}
}

func TestIngestLocation_InvalidSpeed(t *testing.T) {
	mockSvc := &mockIngestService{}
	handler := NewLocationHandler(mockSvc)

	speeds := []float64{-0.1, -10.0, 350.1, 500.0}
	for _, speed := range speeds {
		t.Run(fmt.Sprintf("Speed_%.1f", speed), func(t *testing.T) {
			body, _ := json.Marshal(models.Location{
				DeviceID:  "dev-1",
				Latitude:  28.6,
				Longitude: 77.2,
				Speed:     speed,
			})
			c, rec := newTestEchoContext(http.MethodPost, "/api/location", bytes.NewReader(body))
			err := handler.IngestLocation(c)
			if err != nil {
				t.Fatalf("unexpected handler error: %v", err)
			}
			if rec.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want %d for speed %f", rec.Code, http.StatusBadRequest, speed)
			}
			if !strings.Contains(rec.Body.String(), "invalid speed") {
				t.Errorf("body = %s, want invalid speed", rec.Body.String())
			}
		})
	}
}

func TestIngestLocation_DefaultTimestamp(t *testing.T) {
	mockSvc := &mockIngestService{}
	handler := NewLocationHandler(mockSvc)

	before := time.Now().Unix()
	payload := `{"device_id": "dev-1", "latitude": 28.6139, "longitude": 77.2090, "speed": 10.0, "timestamp": 0}`
	c, rec := newTestEchoContext(http.MethodPost, "/api/location", strings.NewReader(payload))

	err := handler.IngestLocation(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	after := time.Now().Unix()
	if mockSvc.lastLoc.Timestamp < before || mockSvc.lastLoc.Timestamp > after {
		t.Errorf("timestamp %d not in expected range [%d, %d]", mockSvc.lastLoc.Timestamp, before, after)
	}
}

func TestIngestLocation_InvalidJSON(t *testing.T) {
	mockSvc := &mockIngestService{}
	handler := NewLocationHandler(mockSvc)

	c, rec := newTestEchoContext(http.MethodPost, "/api/location", strings.NewReader("not-json"))
	err := handler.IngestLocation(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	if !strings.Contains(rec.Body.String(), "invalid request body") {
		t.Errorf("body = %s, want invalid request body", rec.Body.String())
	}
}

func TestIngestLocation_ServiceError(t *testing.T) {
	mockSvc := &mockIngestService{
		ingestFn: func(ctx context.Context, loc *models.Location) error {
			return fmt.Errorf("database connection pool exhausted")
		},
	}
	handler := NewLocationHandler(mockSvc)

	payload := `{"device_id": "dev-1", "latitude": 28.6139, "longitude": 77.2090, "speed": 10.0}`
	c, rec := newTestEchoContext(http.MethodPost, "/api/location", strings.NewReader(payload))

	err := handler.IngestLocation(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "internal server error") {
		t.Errorf("body = %s, want internal server error", rec.Body.String())
	}
}
