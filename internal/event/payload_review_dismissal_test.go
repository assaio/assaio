package event

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestReviewDismissalValidate(t *testing.T) {
	at := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	base := ReviewDismissal{Number: 1, PreviousState: "approved", DismissedAt: at}
	for _, tt := range []struct {
		name string
		edit func(*ReviewDismissal)
		want string
	}{
		{"null review", func(*ReviewDismissal) {}, ""},
		{"joined review", func(p *ReviewDismissal) {
			p.ReviewID, p.Commit, p.SubmittedAt = "PRR_1", mergeCommit, at.Add(-time.Hour)
		}, ""},
		{"pending source state", func(p *ReviewDismissal) { p.PreviousState = "pending" }, ""},
		{"changes requested", func(p *ReviewDismissal) { p.PreviousState = "changes_requested" }, ""},
		{"unknown state", func(p *ReviewDismissal) { p.PreviousState = "round_2" }, "unknown previous review state"},
		{"missing state", func(p *ReviewDismissal) { p.PreviousState = "" }, "unknown previous review state"},
		{"missing number", func(p *ReviewDismissal) { p.Number = 0 }, "not positive"},
		{"missing actual time", func(p *ReviewDismissal) { p.DismissedAt = time.Time{} }, "no actual dismissal time"},
		{"commit without join", func(p *ReviewDismissal) { p.Commit = mergeCommit }, "without a review id"},
		{"submission without join", func(p *ReviewDismissal) { p.SubmittedAt = at }, "without a review id"},
		{"text id", func(p *ReviewDismissal) { p.ReviewID = "review body" }, "no usable id"},
		{"oversized id", func(p *ReviewDismissal) { p.ReviewID = strings.Repeat("a", 129) }, "no usable id"},
		{"bad commit", func(p *ReviewDismissal) { p.ReviewID, p.Commit = "PRR_1", "abc" }, "not a commit hash"},
		{"sha256", func(p *ReviewDismissal) { p.ReviewID, p.Commit = "PRR_1", strings.Repeat("a", 64) }, ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			p := base
			tt.edit(&p)
			assertErr(t, p.validate(), tt.want)
		})
	}
}

func TestTimelinePopulationValidate(t *testing.T) {
	for _, tt := range []struct {
		name string
		p    TimelinePopulation
		want string
	}{
		{"whole timeline with no dismissals", TimelinePopulation{TotalCount: 8}, ""},
		{"one complete dismissal", TimelinePopulation{TotalCount: 8, FilteredCount: 1, PageCount: 1, Listed: 1}, ""},
		{"bounded dismissal slice", TimelinePopulation{TotalCount: 300, FilteredCount: 101, PageCount: 100, Listed: 100, HasPreviousPage: true}, ""},
		{"negative total", TimelinePopulation{TotalCount: -1}, "negative"},
		{"negative filtered", TimelinePopulation{FilteredCount: -1}, "negative"},
		{"negative page", TimelinePopulation{PageCount: -1}, "negative"},
		{"negative listed", TimelinePopulation{Listed: -1}, "negative"},
		{"filtered exceeds whole", TimelinePopulation{TotalCount: 1, FilteredCount: 2, PageCount: 2, Listed: 2}, "inconsistent"},
		{"page exceeds filtered", TimelinePopulation{TotalCount: 3, FilteredCount: 1, PageCount: 2, Listed: 2}, "inconsistent"},
		{"listed differs from page", TimelinePopulation{TotalCount: 3, FilteredCount: 2, PageCount: 2, Listed: 1}, "inconsistent"},
		{"short under fixed bound", TimelinePopulation{TotalCount: 3, FilteredCount: 2, PageCount: 1, Listed: 1, HasPreviousPage: true}, "last:100"},
		{"false truncation", TimelinePopulation{TotalCount: 3, HasPreviousPage: true}, "last:100"},
		{"hidden truncation", TimelinePopulation{TotalCount: 101, FilteredCount: 101, PageCount: 100, Listed: 100}, "last:100"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			assertErr(t, tt.p.validate(), tt.want)
			pr := PullRequest{Number: 1, State: ChangeOpen, ReviewDismissals: &tt.p}
			assertErr(t, pr.validate(), tt.want)
		})
	}
}

func TestReviewDismissalJSONPreservesUnavailableAndNullJoin(t *testing.T) {
	p := PullRequest{Number: 1, State: ChangeOpen}
	b, err := json.Marshal(p)
	if err != nil || !strings.Contains(string(b), `"reviewDismissals":null`) {
		t.Fatalf("unavailable timeline = %s, %v", b, err)
	}
	d := ReviewDismissal{Number: 1, PreviousState: "approved", DismissedAt: time.Unix(1, 0)}
	b, err = json.Marshal(d)
	if err != nil || strings.Contains(string(b), "reviewId") || strings.Contains(string(b), "submittedAt") || strings.Contains(string(b), "commit") {
		t.Fatalf("null source join = %s, %v", b, err)
	}
	e := validEvent()
	e.Type, e.Grain, e.Privacy, e.Payload = TypeReviewDismissal, GrainReview, LocalOnly, d
	assertErr(t, e.Validate(), "")
}
