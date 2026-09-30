package attribution

import (
	"fmt"
	"io"
	"strings"
	"time"
)

// RenderText writes the human projection of the same document JSON exposes.
func RenderText(w io.Writer, doc *Document) error {
	project := doc.Project
	if project == "" {
		project = "—"
	}
	if _, err := fmt.Fprintf(w, "Evidence · %s   algorithm: %s   following window: %dh\n",
		project, doc.Algorithm, doc.MaxGapSeconds/3600); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w,
		"  population: %d session(s) for this project · %d other-project · %d project-unknown · %d repository-unresolved\n",
		doc.Summary.Population, doc.OtherProjects, doc.ProjectUnknown, doc.IdentityUnresolved); err != nil {
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
	if err := renderForge(w, &doc.Forge); err != nil {
		return err
	}
	if err := renderChanges(w, doc.Changes, doc.Since); err != nil {
		return err
	}

	for i := range doc.Results {
		if err := renderResult(w, &doc.Results[i], doc.Changes != nil); err != nil {
			return err
		}
	}
	if err := renderLinked(w, doc.Changes); err != nil {
		return err
	}
	_, err := fmt.Fprintln(w, `
  Observation only: project and time proximity produce candidates, confidence and abstention;
  they do not prove that an AI session caused a commit or measure a person's performance.
  Local-only and content-free: no prompt, response, code, diff, commit message or branch name is stored.`)
	if err != nil {
		return err
	}
	return renderOutcomeScope(w, doc.Changes != nil)
}

func renderResult(w io.Writer, result *Result, withChanges bool) error {
	if _, err := fmt.Fprintf(w, "\n  %s  %s  confidence=%s method=%s provenance=%s",
		result.Session.ID, result.Status, result.Confidence, result.Method, result.Provenance); err != nil {
		return err
	}
	if result.Reason != "" {
		if _, err := fmt.Fprintf(w, " reason=%s", result.Reason); err != nil {
			return err
		}
	}
	if result.Change != 0 {
		if _, err := fmt.Fprintf(w, " one-pull-request=#%d", result.Change); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}
	for i := range result.Candidates {
		if err := renderCandidate(w, "candidate", &result.Candidates[i], withChanges); err != nil {
			return err
		}
	}
	for i := range result.Alternatives {
		if err := renderCandidate(w, "alternative", &result.Alternatives[i], withChanges); err != nil {
			return err
		}
	}
	return nil
}

func renderCandidate(w io.Writer, label string, candidate *Candidate, withChanges bool) error {
	relation := "inside session"
	switch {
	case candidate.Relation == relationLanded && candidate.GapSeconds == 0:
		relation = "landed by forge during session"
	case candidate.Relation == relationLanded:
		relation = fmt.Sprintf("landed by forge %ds after session", candidate.GapSeconds)
	case candidate.Relation == relationFollowing:
		relation = fmt.Sprintf("%ds after session", candidate.GapSeconds)
	}
	judged := candidate.EvidenceAt.Format(time.RFC3339)
	if !candidate.EvidenceAt.Equal(candidate.OccurredAt) {
		judged += " (written; committed " + candidate.OccurredAt.Format(time.RFC3339) + ")"
	}
	_, err := fmt.Fprintf(w,
		"    %s %s · %s · %s · source=%s time=%s provenance=%s privacy=%s · files=%s%s\n",
		label, shortID(candidate.CommitID), judged, relation,
		candidate.Source.Name, candidate.TimeSource, candidate.Provenance, candidate.Privacy,
		fileCategories(candidate), changeLinks(candidate, withChanges))
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

// renderForge states what forge detection saw, always -- a reader has to be able to tell "no
// forge commit" from "a forge this build does not recognise" -- and, when it recognised anything,
// what coverage means on such history.
func renderForge(w io.Writer, f *Forge) error {
	if !f.Recognised() {
		_, err := fmt.Fprintf(w,
			"  forge (%s only): none recognised; merges by other forges are judged as local commits\n", f.Detection)
		return err
	}
	if _, err := fmt.Fprintf(w,
		"  forge (%s only): %d landed · %d rebased · %d merge commit(s) excluded · %d session(s) unmatched with a later landing beyond their following window\n",
		f.Detection, f.LandedCommits, f.RebasedCommits, f.MergeCommits, f.LaterLanding); err != nil {
		return err
	}
	_, err := fmt.Fprintln(w, "  On forge-merged history, coverage measures what commit timing can link, not what shipped.")
	return err
}
