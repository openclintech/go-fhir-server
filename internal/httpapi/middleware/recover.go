package middleware

import (
	"log"
	"net/http"

	"go-fhir-server/internal/fhir"
	"go-fhir-server/internal/httpapi/respond"
)

func Recover(l *log.Logger) func(http.Handler) http.Handler {
	if l == nil {
		l = log.Default()
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					l.Printf("panic: %v", rec)
					respond.JSON(w, http.StatusInternalServerError, fhir.OperationOutcome("internal server error"), "application/fhir+json")
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
