package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	store := newPatientStore()

	mux := http.NewServeMux()

	// "health" (use /ping to avoid the /healthz weirdness you saw)
	mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			httpError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"pong": true,
			"time": time.Now().UTC().Format(time.RFC3339),
		}, "application/json")
	})

	// Patient CRUD + search
	mux.HandleFunc("/fhir/Patient", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			handleCreatePatient(store, w, r)
		case http.MethodGet:
			handleSearchPatients(store, w, r)
		default:
			w.Header().Set("Allow", strings.Join([]string{http.MethodPost, http.MethodGet}, ", "))
			httpError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	})

	mux.HandleFunc("/fhir/Patient/", func(w http.ResponseWriter, r *http.Request) {
		// /fhir/Patient/{id}
		id := strings.TrimPrefix(r.URL.Path, "/fhir/Patient/")
		id = strings.Trim(id, "/")
		if id == "" {
			httpError(w, http.StatusNotFound, "not found")
			return
		}
		if !fhirIDRe.MatchString(id) {
			httpError(w, http.StatusBadRequest, "invalid id")
			return
		}

		switch r.Method {
		case http.MethodGet:
			handleReadPatient(store, id, w, r)
		case http.MethodPut:
			handleUpdatePatient(store, id, w, r)
		case http.MethodDelete:
			handleDeletePatient(store, id, w, r)
		default:
			w.Header().Set("Allow", strings.Join([]string{http.MethodGet, http.MethodPut, http.MethodDelete}, ", "))
			httpError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	})

	// Root: convenience
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"ok": true,
			"paths": []string{
				"/ping",
				"/fhir/Patient (POST create, GET search)",
				"/fhir/Patient/{id} (GET read, PUT update, DELETE delete)",
			},
		}, "application/json")
	})

	addr := ":" + port
	log.Printf("listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

// -------------------- Handlers --------------------

func handleCreatePatient(store *patientStore, w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	patient, ok := decodePatient(w, r)
	if !ok {
		return
	}

	// assign id + meta
	if _, exists := patient["id"]; !exists {
		patient["id"] = newFHIRID()
	} else if !isNonEmptyString(patient["id"]) {
		httpError(w, http.StatusBadRequest, "id must be a non-empty string when provided")
		return
	}

	id := patient["id"].(string)
	if !fhirIDRe.MatchString(id) {
		httpError(w, http.StatusBadRequest, "id is not a valid FHIR id")
		return
	}

	ensureMeta(patient, 1)

	store.put(id, patient)

	w.Header().Set("Location", "/fhir/Patient/"+id)
	writeJSON(w, http.StatusCreated, patient, "application/fhir+json")
}

func handleReadPatient(store *patientStore, id string, w http.ResponseWriter, r *http.Request) {
	p, ok := store.get(id)
	if !ok {
		httpError(w, http.StatusNotFound, "not found")
		return
	}
	writeJSON(w, http.StatusOK, p, "application/fhir+json")
}

func handleUpdatePatient(store *patientStore, id string, w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	patient, ok := decodePatient(w, r)
	if !ok {
		return
	}

	// force body id to match path id (or set it)
	if v, exists := patient["id"]; exists {
		if !isNonEmptyString(v) || v.(string) != id {
			httpError(w, http.StatusBadRequest, "body.id must match URL id")
			return
		}
	} else {
		patient["id"] = id
	}

	// bump versionId if exists; otherwise start at 1
	nextVersion := store.nextVersion(id)
	ensureMeta(patient, nextVersion)

	store.put(id, patient)
	writeJSON(w, http.StatusOK, patient, "application/fhir+json")
}

func handleDeletePatient(store *patientStore, id string, w http.ResponseWriter, r *http.Request) {
	if !store.delete(id) {
		httpError(w, http.StatusNotFound, "not found")
		return
	}
	// FHIR delete often returns 204
	w.WriteHeader(http.StatusNoContent)
}

func handleSearchPatients(store *patientStore, w http.ResponseWriter, r *http.Request) {
	// Minimal: return all patients as a Bundle
	all := store.list()

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

	writeJSON(w, http.StatusOK, bundle, "application/fhir+json")
}

// -------------------- Decode / Validate --------------------

func decodePatient(w http.ResponseWriter, r *http.Request) (map[string]any, bool) {
	dec := json.NewDecoder(r.Body)

	// For speed, keep permissive. If you want strict later, re-enable:
	// dec.DisallowUnknownFields()

	var payload map[string]any
	if err := dec.Decode(&payload); err != nil {
		httpError(w, http.StatusBadRequest, "invalid JSON body")
		return nil, false
	}

	// Must be a Patient
	rt, ok := payload["resourceType"]
	if !ok || !isNonEmptyString(rt) || rt.(string) != "Patient" {
		httpError(w, http.StatusBadRequest, "resourceType must be 'Patient'")
		return nil, false
	}

	return payload, true
}

func ensureMeta(resource map[string]any, version int) {
	meta, _ := resource["meta"].(map[string]any)
	if meta == nil {
		meta = map[string]any{}
		resource["meta"] = meta
	}
	meta["versionId"] = intToString(version)
	meta["lastUpdated"] = time.Now().UTC().Format(time.RFC3339)
}

// -------------------- Helpers --------------------

func writeJSON(w http.ResponseWriter, status int, v any, contentType string) {
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func httpError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]any{
		"resourceType": "OperationOutcome",
		"issue": []map[string]any{
			{
				"severity": "error",
				"code":     "processing",
				"details":  map[string]any{"text": message},
			},
		},
	}, "application/fhir+json")
}

var fhirIDRe = regexp.MustCompile(`^[A-Za-z0-9\-\.]{1,64}$`)

func newFHIRID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "id-" + time.Now().UTC().Format("20060102150405")
	}
	return hex.EncodeToString(b)
}

func isNonEmptyString(v any) bool {
	s, ok := v.(string)
	return ok && strings.TrimSpace(s) != ""
}

func intToString(n int) string {
	// tiny helper to avoid fmt import
	return strings.TrimSpace(strings.ReplaceAll(strings.Trim(strings.Repeat(" ", 0), " "), "", "")) + itoa(n)
}

// minimal itoa (avoids fmt/strconv import)
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	sign := ""
	if n < 0 {
		sign = "-"
		n = -n
	}
	var b [32]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return sign + string(b[i:])
}

// -------------------- Store --------------------

type patientStore struct {
	mu       sync.RWMutex
	data     map[string]map[string]any
	versions map[string]int
}

func newPatientStore() *patientStore {
	return &patientStore{
		data:     make(map[string]map[string]any),
		versions: make(map[string]int),
	}
}

func (s *patientStore) put(id string, patient map[string]any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[id] = deepCopy(patient)
	// keep version map in sync if meta.versionId present
	if meta, ok := patient["meta"].(map[string]any); ok {
		if v, ok := meta["versionId"].(string); ok {
			// best effort parse
			if vv := atoi(v); vv > 0 {
				s.versions[id] = vv
			}
		}
	}
}

func (s *patientStore) get(id string) (map[string]any, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.data[id]
	if !ok {
		return nil, false
	}
	return deepCopy(v), true
}

func (s *patientStore) delete(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.data[id]; !ok {
		return false
	}
	delete(s.data, id)
	delete(s.versions, id)
	return true
}

func (s *patientStore) list() []map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]map[string]any, 0, len(s.data))
	for _, v := range s.data {
		out = append(out, deepCopy(v))
	}
	return out
}

func (s *patientStore) nextVersion(id string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.versions[id]++
	if s.versions[id] == 0 {
		s.versions[id] = 1
	}
	return s.versions[id]
}

func deepCopy(m map[string]any) map[string]any {
	b, _ := json.Marshal(m)
	var out map[string]any
	_ = json.Unmarshal(b, &out)
	return out
}

func atoi(s string) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	sign := 1
	if strings.HasPrefix(s, "-") {
		sign = -1
		s = strings.TrimPrefix(s, "-")
	}
	n := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c < '0' || c > '9' {
			return 0
		}
		n = n*10 + int(c-'0')
	}
	return sign * n
}
