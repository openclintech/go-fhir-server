package memory

import (
	"encoding/json"
	"sync"
)

type PatientStore struct {
	mu       sync.RWMutex
	data     map[string]map[string]any
	versions map[string]int
}

func NewPatientStore() *PatientStore {
	return &PatientStore{
		data:     make(map[string]map[string]any),
		versions: make(map[string]int),
	}
}

func (s *PatientStore) Put(id string, patient map[string]any) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.data[id] = deepCopy(patient)

	// Ensure a newly created resource starts at version 1.
	// Updates will bump via NextVersion().
	if _, ok := s.versions[id]; !ok {
		s.versions[id] = 1
	}

	return nil
}

func (s *PatientStore) Get(id string) (map[string]any, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.data[id]
	if !ok {
		return nil, false, nil
	}
	return deepCopy(v), true, nil
}

func (s *PatientStore) Delete(id string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.data[id]; !ok {
		return false, nil
	}
	delete(s.data, id)
	delete(s.versions, id)
	return true, nil
}

func (s *PatientStore) List() ([]map[string]any, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]map[string]any, 0, len(s.data))
	for _, v := range s.data {
		out = append(out, deepCopy(v))
	}
	return out, nil
}

func (s *PatientStore) NextVersion(id string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// If it hasn't been set yet, start at 1 then bump to 2 for first update.
	if _, ok := s.versions[id]; !ok {
		s.versions[id] = 1
	}

	s.versions[id]++
	return s.versions[id], nil
}

func deepCopy(m map[string]any) map[string]any {
	b, _ := json.Marshal(m)
	var out map[string]any
	_ = json.Unmarshal(b, &out)
	return out
}
