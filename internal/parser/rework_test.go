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
	if m["a.go"] != 6 {
		t.Fatalf("unreworked[a.go] = %d, want 6 (the 4 lines already counted as rework are spent)", m["a.go"])
	}
}

// TestReworkNeverExceedsTheAdditionsItUndoes is the invariant the cap exists for: two
// removals cannot each claim the same added lines, so rework can never exceed the file's
// additions and the rate can never exceed 100%.
func TestReworkNeverExceedsTheAdditionsItUndoes(t *testing.T) {
	m := map[string]int64{}
	Rework(m, "d.go", 3, 0)
	first := Rework(m, "d.go", 0, 10)
	second := Rework(m, "d.go", 0, 10)
	if first+second != 3 {
		t.Fatalf("rework total = %d+%d, want 3 in total (only 3 lines were ever added to undo)", first, second)
	}
}

// TestReworkBudgetIsRefilledByLaterAdditions covers the other half: consuming the budget
// must not make a file permanently immune to rework once it is written to again.
func TestReworkBudgetIsRefilledByLaterAdditions(t *testing.T) {
	m := map[string]int64{}
	Rework(m, "e.go", 5, 0)
	if r := Rework(m, "e.go", 0, 5); r != 5 {
		t.Fatalf("first removal = %d, want 5", r)
	}
	Rework(m, "e.go", 7, 0)
	if r := Rework(m, "e.go", 0, 7); r != 7 {
		t.Fatalf("removal after a fresh 7-line addition = %d, want 7", r)
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
