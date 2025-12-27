package fhir

import "regexp"

// FHIR id: [A-Za-z0-9\-\.]{1,64}
var IDRe = regexp.MustCompile(`^[A-Za-z0-9\-\.]{1,64}$`)
