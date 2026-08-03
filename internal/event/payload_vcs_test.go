package event

import "testing"

func TestCommitValidate(t *testing.T) {
	tests := []struct {
		name    string
		commit  Commit
		wantErr string
	}{
		{
			"an ordinary commit",
			Commit{
				Parents: 1, FilesChanged: 2, LinesAdded: 10, LinesRemoved: 3,
				Files: FileCategories{Source: 1, Test: 1},
			},
			"",
		},
		{
			"an empty commit is still an observation: it happened",
			Commit{Parents: 1},
			"",
		},
		{
			"a merge carries no line counts of its own",
			Commit{Parents: 2},
			"",
		},
		{"negative added", Commit{Parents: 1, LinesAdded: -1}, "linesAdded is negative"},
		{"negative parents", Commit{Parents: -1}, "parents is negative"},
		{
			"a category split that loses a file is not a split",
			Commit{Parents: 1, FilesChanged: 3, Files: FileCategories{Source: 1}},
			"categories account for 1 file",
		},
		{
			"nor is one that invents a file",
			Commit{Parents: 1, FilesChanged: 1, Files: FileCategories{Source: 1, Other: 1}},
			"categories account for 2 file",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assertErr(t, tc.commit.validate(), tc.wantErr)
		})
	}
}

// A commit is neither a turn nor a session. Reusing an AI grain for it would let a consumer
// average two things that are not the same unit, which is what the field exists to prevent.
func TestACommitObservationCarriesTheCommitGrain(t *testing.T) {
	e := validEvent()
	e.Type, e.ID, e.Grain, e.Privacy = TypeCommit, "abc123", GrainCommit, LocalOnly
	e.Source = Source{Name: "git", Build: "v0.7.0"}
	e.Payload = Commit{Parents: 1, FilesChanged: 1, LinesAdded: 4, Files: FileCategories{Source: 1}}
	if err := e.Validate(); err != nil {
		t.Fatalf("a commit observation was rejected: %v", err)
	}
}
