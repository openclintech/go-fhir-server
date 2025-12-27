package fhir

import (
	"strconv"
	"time"
)

func EnsureMeta(resource map[string]any, version int) {
	meta, _ := resource["meta"].(map[string]any)
	if meta == nil {
		meta = map[string]any{}
		resource["meta"] = meta
	}
	meta["versionId"] = strconv.Itoa(version)
	meta["lastUpdated"] = time.Now().UTC().Format(time.RFC3339)
}
