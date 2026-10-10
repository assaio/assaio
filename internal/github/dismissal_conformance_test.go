package github

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/event"
)

func TestCapturedDismissalsRetainActualTransitionsAndFilteredPopulations(t *testing.T) {
	body, err := os.ReadFile("testdata/b232/production-dismissals.json")
	if err != nil {
		t.Fatal(err)
	}
	var response struct {
		Data map[string]map[string]node `json:"data"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		t.Fatal(err)
	}
	observed := time.Date(2026, 10, 10, 21, 30, 48, 0, time.UTC)
	for _, tt := range []struct {
		repository, key, id string
		number, total       int64
	}{
		{"docs", "positive", "RDE_lADOC01lZ88AAAABVPzQjs8AAAAHmFr34w", 46200, 23},
		{"vscode", "positive1", "RDE_lADOAn8RLM8AAAABVQAy7s8AAAAHlJ3RHQ", 339957, 24},
		{"vscode", "positive2", "RDE_lADOAn8RLM8AAAABWM2Nss8AAAAHqair3w", 340803, 22},
		{"cli", "negative", "", 14640, 25},
	} {
		t.Run(tt.repository+"/"+tt.key, func(t *testing.T) {
			n := response.Data[tt.repository][tt.key]
			population, events, err := dismissalObservations(&n, event.Source{Name: sourceName}, "p", observed)
			if err != nil {
				t.Fatal(err)
			}
			if population == nil || n.Number != tt.number || population.TotalCount != tt.total || population.HasPreviousPage {
				t.Fatalf("real filtered population = %+v for PR %d", population, n.Number)
			}
			if tt.id == "" {
				if population.FilteredCount != 0 || population.PageCount != 0 || population.Listed != 0 || len(events) != 0 {
					t.Fatal("nonempty whole timeline became a nonempty dismissal selection")
				}
				return
			}
			if population.FilteredCount != 1 || population.PageCount != 1 || population.Listed != 1 || len(events) != 1 || events[0].ID != tt.id {
				t.Fatalf("real positive dismissal = %+v, %+v", population, events)
			}
			e := events[0]
			p := e.Payload.(event.ReviewDismissal)
			raw := n.ReviewDismissals.Nodes[0]
			if err := e.Validate(); err != nil {
				t.Fatal(err)
			}
			if p.PreviousState != "approved" || p.ReviewID != raw.Review.ID || p.Commit != raw.Review.Commit.OID ||
				!p.SubmittedAt.Equal(raw.Review.SubmittedAt) || !p.DismissedAt.Equal(raw.CreatedAt) ||
				!e.OccurredAt.Equal(raw.CreatedAt) || !e.ObservedAt.Equal(observed) || e.TimeSource != event.TimeStated ||
				e.Privacy != event.LocalOnly || e.Provenance != event.Parsed || e.Subject.Member != "" || e.Subject.Session != "" {
				t.Fatalf("real transition lost its source, clocks or privacy: %+v", e)
			}
		})
	}
}

func TestCapturedZeroSliceIsTruncationRatherThanAbsence(t *testing.T) {
	body, err := os.ReadFile("testdata/b232/github-docs-dismissal-truncation.json")
	if err != nil {
		t.Fatal(err)
	}
	var response struct {
		Data struct {
			Repository struct {
				PullRequest struct {
					Zero dismissalConnection `json:"zero"`
					One  dismissalConnection `json:"one"`
				} `json:"pullRequest"`
			} `json:"repository"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		t.Fatal(err)
	}
	p := response.Data.Repository.PullRequest
	if *p.Zero.TotalCount != 23 || *p.Zero.FilteredCount != 1 || *p.Zero.PageCount != 0 ||
		!*p.Zero.PageInfo.HasPreviousPage || len(p.Zero.Nodes) != 0 || *p.One.FilteredCount != 1 ||
		*p.One.PageCount != 1 || *p.One.PageInfo.HasPreviousPage || len(p.One.Nodes) != 1 {
		t.Fatal("actual source counts do not distinguish filtered absence from a zero-sized slice")
	}
	// last:0 is an investigative source proof, not the production last:100 contract.
	if _, err := dismissalPopulation(&p.Zero); err == nil {
		t.Fatal("an investigative zero-sized slice satisfied the production read bound")
	}
}

func TestB232CaptureManifestMatchesSourceBytes(t *testing.T) {
	body, err := os.ReadFile("testdata/b232/captures.json")
	if err != nil {
		t.Fatal(err)
	}
	type digest struct {
		File, SHA256 string
	}
	var manifest struct {
		Captures []digest
		Queries  []digest
	}
	if err := json.Unmarshal(body, &manifest); err != nil {
		t.Fatal(err)
	}
	if len(manifest.Captures) != 11 || len(manifest.Queries) != 9 {
		t.Fatal("finite source corpus lost a capture or query")
	}
	for _, d := range append(manifest.Captures, manifest.Queries...) {
		content, err := os.ReadFile("testdata/b232/" + d.File)
		if err != nil {
			t.Fatal(err)
		}
		if got := fmt.Sprintf("%x", sha256.Sum256(content)); got != d.SHA256 {
			t.Fatalf("source bytes changed for %s: %s", d.File, got)
		}
	}
}
