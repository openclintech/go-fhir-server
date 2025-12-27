package memory

import "testing"

func TestPatientStore_PutGetDeleteList(t *testing.T) {
	s := NewPatientStore()

	p := map[string]any{"resourceType": "Patient", "id": "abc"}
	if err := s.Put("abc", p); err != nil {
		t.Fatalf("put err: %v", err)
	}

	got, ok, err := s.Get("abc")
	if err != nil {
		t.Fatalf("get err: %v", err)
	}
	if !ok {
		t.Fatalf("expected ok=true")
	}
	if got["id"] != "abc" {
		t.Fatalf("expected id abc, got %v", got["id"])
	}

	all, err := s.List()
	if err != nil {
		t.Fatalf("list err: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("expected 1 patient, got %d", len(all))
	}

	ok, err = s.Delete("abc")
	if err != nil {
		t.Fatalf("delete err: %v", err)
	}
	if !ok {
		t.Fatalf("expected ok=true on delete")
	}
}
