package app

import (
	"log"
	"net/http"
	"time"

	"go-fhir-server/internal/httpapi/middleware"
	"go-fhir-server/internal/storage"
)

type Deps struct {
	PatientStore storage.PatientStore
	Logger       *log.Logger
}

func New(d Deps) http.Handler {
	mux := http.NewServeMux()

	// Routes
	registerRoutes(mux, d)

	// Middlewares (outermost -> innermost)
	var h http.Handler = mux
	h = middleware.Recover(d.Logger)(h)
	h = middleware.RequestID()(h)
	h = middleware.Logging(d.Logger)(h)

	// Very reasonable default server-side timeouts can also be set on http.Server
	// (kept in main.go if you want). For now we keep a simple handler timeout:
	h = http.TimeoutHandler(h, 15*time.Second, `{"resourceType":"OperationOutcome","issue":[{"severity":"error","code":"timeout","details":{"text":"request timed out"}}]}`)

	return h
}
