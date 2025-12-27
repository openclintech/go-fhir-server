package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-fhir-server/internal/httpapi/handlers"
	"go-fhir-server/internal/storage/memory"
)

// readJSON is a tiny helper to decode a JSON response body into a map.
func readJSON(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()

	var m map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &m); err != nil {
		t.Fatalf("invalid JSON response: %v (body=%s)", err, rec.Body.String())
	}
	return m
}

// requireString pulls a string out of a map and fails the test if it's missing/not a string.
func requireString(t *testing.T, m map[string]any, key string) string {
	t.Helper()

	v, ok := m[key]
	if !ok {
		t.Fatalf("expected key %q to exist", key)
	}
	s, ok := v.(string)
	if !ok || s == "" {
		t.Fatalf("expected %q to be a non-empty string, got %T (%v)", key, v, v)
	}
	return s
}

// requireMap pulls a nested map out of a map and fails the test if it's missing/not a map.
func requireMap(t *testing.T, m map[string]any, key string) map[string]any {
	t.Helper()

	v, ok := m[key]
	if !ok {
		t.Fatalf("expected key %q to exist", key)
	}
	nested, ok := v.(map[string]any)
	if !ok || nested == nil {
		t.Fatalf("expected %q to be a map, got %T (%v)", key, v, v)
	}
	return nested
}

func TestPatient_CreateReadUpdateDelete(t *testing.T) {
	store := memory.NewPatientStore()
	h := handlers.Patient(store)

	// ---- CREATE (POST /fhir/Patient)
	{
		createBody := `{"resourceType":"Patient","name":[{"family":"Doe","given":["Jane"]}]}`
		req := httptest.NewRequest(http.MethodPost, "/fhir/Patient", bytes.NewBufferString(createBody))
		req.Header.Set("Content-Type", "application/fhir+json")
		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("create status=%d body=%s", rec.Code, rec.Body.String())
		}
		if ct := rec.Header().Get("Content-Type"); ct == "" {
			t.Fatalf("expected Content-Type to be set")
		}
		if loc := rec.Header().Get("Location"); loc == "" {
			t.Fatalf("expected Location header")
		}

		created := readJSON(t, rec)
		id := requireString(t, created, "id")

		meta := requireMap(t, created, "meta")
		_ = requireString(t, meta, "versionId") // just ensure it exists and is a string
		_ = requireString(t, meta, "lastUpdated")

		// ---- READ (GET /fhir/Patient/{id})
		req = httptest.NewRequest(http.MethodGet, "/fhir/Patient/"+id, nil)
		rec = httptest.NewRecorder()

		h.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("read status=%d body=%s", rec.Code, rec.Body.String())
		}

		read := readJSON(t, rec)
		if got := requireString(t, read, "id"); got != id {
			t.Fatalf("read id mismatch: got=%s want=%s", got, id)
		}

		// ---- UPDATE (PUT /fhir/Patient/{id})
		// We don't enforce exact bump semantics yet; we just require meta.versionId exists.
		updateBody := `{"resourceType":"Patient","id":"` + id + `","active":true}`
		req = httptest.NewRequest(http.MethodPut, "/fhir/Patient/"+id, bytes.NewBufferString(updateBody))
		req.Header.Set("Content-Type", "application/fhir+json")
		rec = httptest.NewRecorder()

		h.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("update status=%d body=%s", rec.Code, rec.Body.String())
		}

		updated := readJSON(t, rec)
		if got := requireString(t, updated, "id"); got != id {
			t.Fatalf("updated id mismatch: got=%s want=%s", got, id)
		}
		umeta := requireMap(t, updated, "meta")
		_ = requireString(t, umeta, "versionId")
		_ = requireString(t, umeta, "lastUpdated")

		// ---- DELETE (DELETE /fhir/Patient/{id})
		req = httptest.NewRequest(http.MethodDelete, "/fhir/Patient/"+id, nil)
		rec = httptest.NewRecorder()

		h.ServeHTTP(rec, req)

		if rec.Code != http.StatusNoContent {
			t.Fatalf("delete status=%d body=%s", rec.Code, rec.Body.String())
		}

		// ---- READ AGAIN -> 404
		req = httptest.NewRequest(http.MethodGet, "/fhir/Patient/"+id, nil)
		rec = httptest.NewRecorder()

		h.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected 404 after delete, got status=%d body=%s", rec.Code, rec.Body.String())
		}
	}
}

func TestPatient_InvalidResourceType(t *testing.T) {
	store := memory.NewPatientStore()
	h := handlers.Patient(store)

	body := `{"resourceType":"Observation"}`
	req := httptest.NewRequest(http.MethodPost, "/fhir/Patient", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/fhir+json")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}

	// Optional: verify it's an OperationOutcome (FHIR-ish error)
	outcome := readJSON(t, rec)
	if rt, _ := outcome["resourceType"].(string); rt != "OperationOutcome" {
		t.Fatalf("expected OperationOutcome, got %v", outcome["resourceType"])
	}
}

func TestPatient_BadIDInPath(t *testing.T) {
	store := memory.NewPatientStore()
	h := handlers.Patient(store)

	// underscore is URL-safe but invalid per your FHIR id regex
	req := httptest.NewRequest(http.MethodGet, "/fhir/Patient/bad_id", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}
}
