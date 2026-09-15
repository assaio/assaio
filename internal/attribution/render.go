package attribution

import (
	"fmt"
	"io"
	"strings"
)

// RenderText writes the human projection of the same document JSON exposes.
func RenderText(w io.Writer, doc *Document) error {
	if _, err := fmt.Fprintf(w, "Evidence · %s   algorithm: %s   following window: %dh\n",
		doc.Project, doc.Algorithm, doc.MaxGapSeconds/3600); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w,
		"  population: %d session(s) for this project · %d other-project · %d project-unknown\n",
		doc.Summary.Population, doc.OtherProjects, doc.ProjectUnknown); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w,
		"  candidate coverage: %.0f%% (%d/%d) · resolved coverage: %.0f%% (%d/%d)\n",
		doc.Summary.CandidateCoverage*100, doc.Summary.CandidateCovered, doc.Summary.Population,
		doc.Summary.ResolvedCoverage*100, doc.Summary.Matched, doc.Summary.Population); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "  outcomes: matched %d · ambiguous %d · unmatched %d",
		doc.Summary.Matched, doc.Summary.Ambiguous, doc.Summary.Unmatched); err != nil {
		return err
	}
	if doc.SkippedCommits > 0 {
		if _, err := fmt.Fprintf(w, " · unreadable commits %d", doc.SkippedCommits); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}

	for i := range doc.Results {
		if err := renderResult(w, &doc.Results[i]); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintln(w, `
  Observation only: project and time proximity produce candidates, confidence and abstention;
  they do not prove that an AI session caused a commit or measure a person's performance.
  Local-only and content-free: no prompt, response, code, diff, commit message or branch name is stored.
  PR, review, CI, merge and downstream outcome correlation are not part of this command.`)
	return err
}

func renderResult(w io.Writer, result *Result) error {
	if _, err := fmt.Fprintf(w, "\n  %s  %s  confidence=%s method=%s provenance=%s",
		result.Session.ID, result.Status, result.Confidence, result.Method, result.Provenance); err != nil {
		return err
	}
	if result.Reason != "" {
		if _, err := fmt.Fprintf(w, " reason=%s", result.Reason); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}
	for i := range result.Candidates {
		if err := renderCandidate(w, "candidate", &result.Candidates[i]); err != nil {
			return err
		}
	}
	for i := range result.Alternatives {
		if err := renderCandidate(w, "alternative", &result.Alternatives[i]); err != nil {
			return err
		}
	}
	return nil
}

func renderCandidate(w io.Writer, label string, candidate *Candidate) error {
	gap := "inside session"
	if candidate.Relation == relationFollowing {
		gap = fmt.Sprintf("%ds after session", candidate.GapSeconds)
	}
	_, err := fmt.Fprintf(w,
		"    %s %s · %s · %s · source=%s time=%s provenance=%s privacy=%s · files=%s\n",
		label, shortID(candidate.CommitID), candidate.OccurredAt.Format("2006-01-02T15:04:05Z07:00"), gap,
		candidate.Source.Name, candidate.TimeSource, candidate.Provenance, candidate.Privacy,
		fileCategories(candidate))
	return err
}

func shortID(id string) string {
	if len(id) <= 12 {
		return id
	}
	return id[:12]
}

func fileCategories(candidate *Candidate) string {
	parts := []string{fmt.Sprintf("total:%d", candidate.FilesChanged)}
	buckets := []struct {
		name  string
		count int64
	}{
		{"source", candidate.Files.Source},
		{"test", candidate.Files.Test},
		{"docs", candidate.Files.Docs},
		{"config", candidate.Files.Config},
		{"generated", candidate.Files.Generated},
		{"other", candidate.Files.Other},
	}
	for _, bucket := range buckets {
		if bucket.count > 0 {
			parts = append(parts, fmt.Sprintf("%s:%d", bucket.name, bucket.count))
		}
	}
	return strings.Join(parts, ",")
}
