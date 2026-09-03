package handlers

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"transit-backend/internal/models"

	"github.com/labstack/echo/v4"
)

// BenchmarkIngestLocation benchmarks the HTTP handler IngestLocation
// using httptest with a mocked IngestService. This measures Echo routing/binding,
// JSON decode, input validation, and JSON response encoding overhead without
// database or network I/O.
func BenchmarkIngestLocation(b *testing.B) {
	mockSvc := &mockIngestService{
		ingestFn: func(ctx context.Context, loc *models.Location) error {
			loc.HexRes9 = "891f5a44a4bffff"
			return nil
		},
	}
	handler := NewLocationHandler(mockSvc)
	e := echo.New()

	payloadBytes := []byte(`{"device_id":"bus-101","latitude":28.6139,"longitude":77.2090,"speed":35.0,"accuracy":4.5,"timestamp":1672531200}`)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/location", bytes.NewReader(payloadBytes))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		if err := handler.IngestLocation(c); err != nil {
			b.Fatalf("IngestLocation failed: %v", err)
		}
	}
}
