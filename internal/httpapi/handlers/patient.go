package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"go-fhir-server/internal/fhir"
	"go-fhir-server/internal/httpapi/respond"
	"go-fhir-server/internal/storage"
)

func Patient(store storage.PatientStore) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Route split:
		// /fhir/Patient          => collection (POST/GET)
		// /fhir/Patient/{id}     => instance (GET/PUT/DELETE)
		path := r.URL.Path

		if path == "/fhir/Patient" {
			switch r.Method {
			case http.MethodPost:
				createPatient(store, w, r)
			case http.MethodGet:
				searchPatients(store, w, r)
			default:
				w.Header().Set("Allow", strings.Join([]string{http.MethodPost, http.MethodGet}, ", "))
				respond.JSON(w, http.StatusMethodNotAllowed, fhir.OperationOutcome("method not allowed"), "application/fhir+json")
			}
			return
		}

		if strings.HasPrefix(path, "/fhir/Patient/") {
			id := strings.TrimPrefix(path, "/fhir/Patient/")
			id = strings.Trim(id, "/")
			if id == "" {
				respond.JSON(w, http.StatusNotFound, fhir.OperationOutcome("not found"), "application/fhir+json")
				return
			}
			if !fhir.IDRe.MatchString(id) {
				respond.JSON(w, http.StatusBadRequest, fhir.OperationOutcome("invalid id"), "application/fhir+json")
				return
			}

			switch r.Method {
			case http.MethodGet:
				readPatient(store, id, w, r)
			case http.MethodPut:
				updatePatient(store, id, w, r)
			case http.MethodDelete:
				deletePatient(store, id, w, r)
			default:
				w.Header().Set("Allow", strings.Join([]string{http.MethodGet, http.MethodPut, http.MethodDelete}, ", "))
				respond.JSON(w, http.StatusMethodNotAllowed, fhir.OperationOutcome("method not allowed"), "application/fhir+json")
			}
			return
		}

		respond.JSON(w, http.StatusNotFound, fhir.OperationOutcome("not found"), "application/fhir+json")
	})
}

func createPatient(store storage.PatientStore, w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	patient, ok := decodePatient(w, r)
	if !ok {
		return
	}

	// assign id + meta
	if _, exists := patient["id"]; !exists {
		patient["id"] = newFHIRID()
	} else if !isNonEmptyString(patient["id"]) {
		respond.JSON(w, http.StatusBadRequest, fhir.OperationOutcome("id must be a non-empty string when provided"), "application/fhir+json")
		return
	}

	id := patient["id"].(string)
	if !fhir.IDRe.MatchString(id) {
		respond.JSON(w, http.StatusBadRequest, fhir.OperationOutcome("id is not a valid FHIR id"), "application/fhir+json")
		return
	}

	fhir.EnsureMeta(patient, 1)

	if err := store.Put(id, patient); err != nil {
		respond.JSON(w, http.StatusInternalServerError, fhir.OperationOutcome("failed to store patient"), "application/fhir+json")
		return
	}

	w.Header().Set("Location", "/fhir/Patient/"+id)
	respond.JSON(w, http.StatusCreated, patient, "application/fhir+json")
}

func readPatient(store storage.PatientStore, id string, w http.ResponseWriter, r *http.Request) {
	p, ok, err := store.Get(id)
	if err != nil {
		respond.JSON(w, http.StatusInternalServerError, fhir.OperationOutcome("storage error"), "application/fhir+json")
		return
	}
	if !ok {
		respond.JSON(w, http.StatusNotFound, fhir.OperationOutcome("not found"), "application/fhir+json")
		return
	}
	respond.JSON(w, http.StatusOK, p, "application/fhir+json")
}

func updatePatient(store storage.PatientStore, id string, w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	patient, ok := decodePatient(w, r)
	if !ok {
		return
	}

	// force body id to match path id (or set it)
	if v, exists := patient["id"]; exists {
		if !isNonEmptyString(v) || v.(string) != id {
			respond.JSON(w, http.StatusBadRequest, fhir.OperationOutcome("body.id must match URL id"), "application/fhir+json")
			return
		}
	} else {
		patient["id"] = id
	}

	nextVersion, err := store.NextVersion(id)
	if err != nil {
		respond.JSON(w, http.StatusInternalServerError, fhir.OperationOutcome("storage error"), "application/fhir+json")
		return
	}

	fhir.EnsureMeta(patient, nextVersion)

	if err := store.Put(id, patient); err != nil {
		respond.JSON(w, http.StatusInternalServerError, fhir.OperationOutcome("failed to store patient"), "application/fhir+json")
		return
	}

	respond.JSON(w, http.StatusOK, patient, "application/fhir+json")
}

func deletePatient(store storage.PatientStore, id string, w http.ResponseWriter, r *http.Request) {
	ok, err := store.Delete(id)
	if err != nil {
		respond.JSON(w, http.StatusInternalServerError, fhir.OperationOutcome("storage error"), "application/fhir+json")
		return
	}
	if !ok {
		respond.JSON(w, http.StatusNotFound, fhir.OperationOutcome("not found"), "application/fhir+json")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func searchPatients(store storage.PatientStore, w http.ResponseWriter, r *http.Request) {
	all, err := store.List()
	if err != nil {
		respond.JSON(w, http.StatusInternalServerError, fhir.OperationOutcome("storage error"), "application/fhir+json")
		return
	}

	entries := make([]map[string]any, 0, len(all))
	for _, p := range all {
		id, _ := p["id"].(string)
		entries = append(entries, map[string]any{
			"fullUrl":  "/fhir/Patient/" + id,
			"resource": p,
		})
	}

	bundle := map[string]any{
		"resourceType": "Bundle",
		"type":         "searchset",
		"total":        len(entries),
		"entry":        entries,
	}

	respond.JSON(w, http.StatusOK, bundle, "application/fhir+json")
}

func decodePatient(w http.ResponseWriter, r *http.Request) (map[string]any, bool) {
	dec := json.NewDecoder(r.Body)

	var payload map[string]any
	if err := dec.Decode(&payload); err != nil {
		respond.JSON(w, http.StatusBadRequest, fhir.OperationOutcome("invalid JSON body"), "application/fhir+json")
		return nil, false
	}

	rt, ok := payload["resourceType"]
	if !ok || !isNonEmptyString(rt) || rt.(string) != "Patient" {
		respond.JSON(w, http.StatusBadRequest, fhir.OperationOutcome("resourceType must be 'Patient'"), "application/fhir+json")
		return nil, false
	}

	return payload, true
}

func isNonEmptyString(v any) bool {
	s, ok := v.(string)
	return ok && strings.TrimSpace(s) != ""
}

func newFHIRID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "id-" + time.Now().UTC().Format("20060102150405")
	}
	return hex.EncodeToString(b)
}
