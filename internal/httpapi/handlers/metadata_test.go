package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"go-fhir-server/internal/httpapi/handlers"
)

func TestMetadata_ReturnsCapabilityStatement(t *testing.T) {
	h := handlers.Metadata()

	req := httptest.NewRequest(http.MethodGet, "/fhir/metadata", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}

	// Minimal sanity checks on body (avoid strict schema for MVP)
	body := rec.Body.String()
	if body == "" {
		t.Fatalf("expected response body")
	}
	if ct := rec.Header().Get("Content-Type"); ct == "" {
		t.Fatalf("expected Content-Type header")
	}
}
