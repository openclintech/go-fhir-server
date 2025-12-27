# Public Go FHIR Server (MVP)

This repository contains a **minimal, public HL7® FHIR® server** implemented in **Go** and deployed on **Google Cloud Run**.

The goal of this project is twofold:

1. **Reference implementation** — demonstrate what a clean, minimal FHIR-style API can look like in Go  
2. **Learning in public** — document and share my process as I build toward a more production-grade FHIR server over time

This is intentionally **small, opinionated, and evolving**.

---

## 🔗 Live Deployment

The server is publicly accessible at:

```
https://go-fhir-server2-724149596628.us-central1.run.app
```

You can interact with it using `curl`, Postman, or any HTTP client.

> ⚠️ **Important note:**  
> This server uses an **in-memory data store**. Data is **not persisted** and may disappear at any time due to Cloud Run scaling, restarts, or deployments.

---

## 🚦 Quick Health Check

```bash
GET /ping
```

Example:

```bash
curl https://go-fhir-server2-724149596628.us-central1.run.app/ping
```

Response:

```json
{
  "pong": true,
  "time": "2025-12-27T20:22:25Z"
}
```

---

## 🧪 Supported FHIR Endpoints (MVP)

### Patient Resource

This server currently supports a minimal subset of **FHIR Patient** operations.

| Operation | Method | Endpoint |
|---------|-------|----------|
| Create Patient | POST | `/fhir/Patient` |
| Read Patient | GET | `/fhir/Patient/{id}` |
| Update Patient | PUT | `/fhir/Patient/{id}` |
| Delete Patient | DELETE | `/fhir/Patient/{id}` |
| Search Patients | GET | `/fhir/Patient` |

All FHIR endpoints use:

```
Content-Type: application/fhir+json
Accept: application/fhir+json
```

---

## 📌 Example Usage

### Create a Patient

```bash
curl -X POST https://go-fhir-server2-724149596628.us-central1.run.app/fhir/Patient \
  -H "Content-Type: application/fhir+json" \
  -H "Accept: application/fhir+json" \
  -d '{
    "resourceType": "Patient",
    "name": [
      { "family": "Doe", "given": ["Jane"] }
    ],
    "gender": "female"
  }'
```

---

### Read a Patient

```bash
GET /fhir/Patient/{id}
```

---

### Update a Patient

```bash
PUT /fhir/Patient/{id}
```

```json
{
  "resourceType": "Patient",
  "id": "{id}",
  "active": true
}
```

---

### Search Patients

```bash
GET /fhir/Patient
```

Returns a `Bundle` of type `searchset`.

---

### Delete a Patient

```bash
DELETE /fhir/Patient/{id}
```

Returns:
- `204 No Content`

---

## 🧠 Design Notes

- This is **not a full FHIR server**
- The API shape follows FHIR conventions where reasonable
- Validation is intentionally permissive
- Persistence is **in-memory only**
- Versioning (`meta.versionId`) exists but is MVP-level

This project prioritizes:
- clarity
- testability
- incremental evolution

over completeness.

---

## 🏗️ Architecture Overview

- **Language:** Go (1.22)
- **HTTP:** `net/http`
- **Storage:** In-memory (per Cloud Run instance)
- **Deployment:** Google Cloud Run (GitHub-connected builds)
- **Testing:** Go test + race detector
- **Formatting & static analysis:** `go fmt`, `go vet`

A simple `Makefile` is used to standardize local checks:

```bash
make check
```

---

## ⚠️ Important Caveats

- Data is **ephemeral**
- Multiple Cloud Run instances do **not share state**
- This server is **not HIPAA compliant**
- Do not send real PHI

This is a **learning and reference project**, not a production clinical system.

---

## 🧭 Roadmap (non-binding)

Possible future additions:
- CapabilityStatement (`/fhir/metadata`)
- Additional FHIR resources (Observation, Condition)
- Search parameters
- ETag / If-Match versioning
- Persistent storage (Firestore / Postgres)
- SMART-on-FHIR–aligned auth patterns

---

## 🤝 Feedback & Learning

This project is intentionally public to:
- share lessons learned
- experiment with FHIR design decisions
- invite constructive feedback

Issues, discussions, and thoughtful suggestions are welcome.

---

## 📄 License

MIT (or update as appropriate)

FHIR® is a registered trademark of HL7 and is used here for educational purposes.
