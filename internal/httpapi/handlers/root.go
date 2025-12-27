package handlers

import (
	"net/http"

	"go-fhir-server/internal/httpapi/respond"
)

func Root() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		respond.JSON(w, http.StatusOK, map[string]any{
			"ok": true,
			"paths": []string{
				"/ping",
				"/fhir/Patient (POST create, GET search)",
				"/fhir/Patient/{id} (GET read, PUT update, DELETE delete)",
			},
		}, "application/json")
	})
}
