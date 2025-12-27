package storage

// PatientStore is intentionally MVP-level.
// Later you can generalize to ResourceStore with resourceType + id.
type PatientStore interface {
	Put(id string, patient map[string]any) error
	Get(id string) (patient map[string]any, ok bool, err error)
	Delete(id string) (ok bool, err error)
	List() ([]map[string]any, error)

	// NextVersion bumps and returns the next versionId for this resource id.
	NextVersion(id string) (int, error)
}
