package fhir

func OperationOutcome(message string) map[string]any {
	return map[string]any{
		"resourceType": "OperationOutcome",
		"issue": []map[string]any{
			{
				"severity": "error",
				"code":     "processing",
				"details":  map[string]any{"text": message},
			},
		},
	}
}
