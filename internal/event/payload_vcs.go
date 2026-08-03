package event

import "fmt"

// Commit is what one commit changed, and none of what it changed. Counts and a category
// split only: no path, no message, no diff, no branch name -- the same guarantee the AI
// payloads make, held by the same test.
type Commit struct {
	// Parents is how many commits this one has: 0 for a root commit, 1 ordinarily, 2 or more
	// for a merge -- which is why a merge's line counts are zero rather than missing.
	Parents      int64 `json:"parents"`
	FilesChanged int64 `json:"filesChanged"`
	LinesAdded   int64 `json:"linesAdded"`
	LinesRemoved int64 `json:"linesRemoved"`
	// Files splits FilesChanged by what kind of file each was, never by which file.
	Files FileCategories `json:"files"`
	// Revert marks a commit the source itself labelled a revert. An undo that does not say
	// so is invisible here, which is why this is an indicator rather than a count.
	Revert bool `json:"revert,omitempty"`
}

// FileCategories counts a commit's changed files per kind. Other is deliberately present:
// without it the split would silently fail to add up to FilesChanged, and a reader would
// take four categories for the whole commit.
type FileCategories struct {
	Test      int64 `json:"test"`
	Source    int64 `json:"source"`
	Docs      int64 `json:"docs"`
	Config    int64 `json:"config"`
	Generated int64 `json:"generated"`
	Other     int64 `json:"other"`
}

// Total is how many files the split accounts for.
func (f FileCategories) Total() int64 {
	return f.Test + f.Source + f.Docs + f.Config + f.Generated + f.Other
}

func (Commit) eventType() string { return TypeCommit }

// validate accepts an empty commit -- one happened, and that is the observation -- but not a
// category split that disagrees with the file count it splits, which would let a consumer
// read four categories as the whole of a commit.
//
//nolint:gocritic // the Payload interface is satisfied by values, so a payload is copied rather than aliased; validated once per commit, not a hot path.
func (c Commit) validate() error {
	if err := nonNegative(map[string]int64{
		"parents": c.Parents, "filesChanged": c.FilesChanged,
		"linesAdded": c.LinesAdded, "linesRemoved": c.LinesRemoved,
		"files.test": c.Files.Test, "files.source": c.Files.Source, "files.docs": c.Files.Docs,
		"files.config": c.Files.Config, "files.generated": c.Files.Generated,
		"files.other": c.Files.Other,
	}); err != nil {
		return err
	}
	if total := c.Files.Total(); total != c.FilesChanged {
		return fmt.Errorf("categories account for %d file(s) but %d changed", total, c.FilesChanged)
	}
	return nil
}
