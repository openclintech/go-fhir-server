package app

import (
	"net/http"

	"go-fhir-server/internal/httpapi/handlers"
)

func registerRoutes(mux *http.ServeMux, d Deps) {
	// Root
	mux.Handle("/", handlers.Root())

	// Health
	mux.Handle("/ping", handlers.Ping())

	// FHIR Metadata
	mux.Handle("/fhir/metadata", handlers.Metadata())

	// FHIR Patient
	patientHandler := handlers.Patient(d.PatientStore)
	mux.Handle("/fhir/Patient", patientHandler)
	mux.Handle("/fhir/Patient/", patientHandler) // /fhir/Patient/{id}
}
