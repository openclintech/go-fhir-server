package main

import (
	"log"
	"net/http"

	"go-fhir-server/internal/app"
	"go-fhir-server/internal/config"
	"go-fhir-server/internal/storage/memory"
)

func main() {
	cfg := config.FromEnv()

	// MVP storage (swap later with Postgres/Firestore/etc.)
	patientStore := memory.NewPatientStore()

	handler := app.New(app.Deps{
		PatientStore: patientStore,
		Logger:       log.Default(),
	})

	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: handler,
	}

	log.Printf("listening on %s", srv.Addr)
	log.Fatal(srv.ListenAndServe())
}
