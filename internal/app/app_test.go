package app_test

import (
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-fhir-server/internal/app"
	"go-fhir-server/internal/storage/memory"
)

func TestApp_RoutesSmoke(t *testing.T) {
	h := app.New(app.Deps{
		PatientStore: memory.NewPatientStore(),
		Logger:       log.Default(),
	})

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
}
