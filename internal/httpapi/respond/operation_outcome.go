package respond

import "net/http"

// OperationOutcome writes a minimal FHIR OperationOutcome error response.
func OperationOutcome(w http.ResponseWriter, status int, message string) {
	JSON(w, status, map[string]any{
		"resourceType": "OperationOutcome",
		"issue": []map[string]any{
			{
				"severity": "error",
				"code":     "processing",
				"details": map[string]any{
					"text": message,
				},
			},
		},
	}, "application/fhir+json")
}
