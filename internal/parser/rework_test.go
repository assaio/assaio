package parser

import "testing"

func TestReworkCountsRemovalOfOwnAddition(t *testing.T) {
	m := map[string]int64{}
	if r := Rework(m, "a.go", 10, 0); r != 0 {
		t.Fatalf("Rework(add 10) = %d, want 0 (nothing removed yet)", r)
	}
	if r := Rework(m, "a.go", 0, 4); r != 4 {
		t.Fatalf("Rework(remove 4 of 10 added) = %d, want 4", r)
	}
	if m["a.go"] != 10 {
		t.Fatalf("addedSoFar[a.go] = %d, want 10 (a removal never decrements it)", m["a.go"])
	}
}

func TestReworkCapsAtPriorAdditionsNotRawRemoval(t *testing.T) {
	m := map[string]int64{}
	Rework(m, "b.go", 3, 0)
	if r := Rework(m, "b.go", 0, 10); r != 3 {
		t.Fatalf("Rework(remove 10 of 3 added) = %d, want 3 (capped)", r)
	}
}

func TestReworkOnUntouchedFileIsZero(t *testing.T) {
	m := map[string]int64{}
	if r := Rework(m, "never-added.go", 0, 5); r != 0 {
		t.Fatalf("Rework on a file never added to = %d, want 0 (deleting pre-existing code is not rework)", r)
	}
}

func TestReworkTracksFilesIndependently(t *testing.T) {
	m := map[string]int64{}
	Rework(m, "a.go", 6, 0)
	Rework(m, "b.go", 3, 0)
	if r := Rework(m, "a.go", 0, 2); r != 2 {
		t.Fatalf("a.go rework = %d, want 2", r)
	}
	if r := Rework(m, "b.go", 0, 5); r != 3 {
		t.Fatalf("b.go rework = %d, want 3 (capped independently of a.go's cap)", r)
	}
}

func TestReworkOwnAdditionNotOffsetBySameCallRemoval(t *testing.T) {
	m := map[string]int64{}
	if r := Rework(m, "c.go", 5, 5); r != 0 {
		t.Fatalf("Rework(add 5, remove 5, same call) = %d, want 0 (added folds in after rework is computed)", r)
	}
}
