package attribution

const (
	renderHistoryCoverage     = "  Historical suite/run read (%s, read %s, back to %s): %d PRs read%s; currently listed commits only, not the full PR lifetime.\n"
	renderRepositoryTotalRead = "  Repository PR total: %s (entire repository, not the window); base read: %d; independent history read: %d.\n"
	renderReviewedRevisions   = "      distinct changes_requested reviewed commit hashes: %s; review population: %s; reason: %s.\n"
	renderExactRoundsWithheld = "      exact requested-changes rounds: withheld; %s.\n"
	renderMergeMethod         = "      merge method: %s; source: %s; reason: %s.\n"
	renderSnapshotRate        = "  %s: %s; numerator: %s; entire named merged PR denominator: %d; eligible review populations: %d; reason: %s. Current snapshot, including after-merge reviews; not a pre-merge rate.\n"
	renderSnapshotRateLabel   = "Named merged PRs with observed changes_requested reviews"
	renderCiRatioWithheld     = "  Comparable PR pipeline CI ratio: withheld; %s.\n"
	renderRawHistoryCoverage  = "      raw historical coverage (listed/total or unavailable): commits %s; suites %s; runs %s. First 100 current commits, last 10 suites per commit, last 5 ALL runs per suite; latest-head contexts are separate.\n"
	renderSourceTimeCaveat    = "      Source times are retained separately from occurrence-time fallbacks. A fallback does not establish an actual review submission time or a complete run sequence.\n"
)

var deliveryReasons = map[string]string{
	"review-states-do-not-define-round-boundaries": "Current review states do not define exact round boundaries.",
	"reviews-unavailable":                          "The review connection is unavailable.",
	"reviews-incomplete":                           "The bounded review connection is incomplete.",
	"review-observation-count-mismatch":            "Review observations do not agree with the connection count.",
	"review-id-unavailable":                        "A review identifier is unavailable.",
	"duplicate-review-observations":                "Review identifiers are not unique.",
	"dismissed-review-state-history-unavailable":   "A dismissed review lacks the state history needed for this count.",
	"pending-review-not-submitted":                 "A pending review is not a submitted review.",
	"review-state-unusable":                        "A review state is unusable for this population.",
	"review-submission-time-unavailable":           "An actual review submission time is unavailable; a fallback cannot replace it.",
	"reviewed-commit-unavailable":                  "A reviewed commit hash is unavailable.",
	"pull-request-not-merged":                      "The pull request is not merged.",
	"merge-commit-unavailable":                     "The merge commit hash is unavailable.",
	"merge-parent-count-unavailable":               "The merge commit parent count is unavailable.",
	"squash-versus-rebase-undetermined":            "One merge parent cannot distinguish squash from rebase.",
	"merge-parent-count-unusable":                  "The merge commit parent count is unusable.",
	"comparable-pr-pipeline-history-unavailable":   "Bounded history on currently listed commits does not establish comparable PR pipeline lifetimes.",
	"no-named-merged-pull-requests":                "There are no named merged pull requests in this population.",
	"ineligible-review-populations":                "At least one named merged PR lacks a nonempty usable review population; the full denominator is retained and the share withheld.",
	"no-submitted-review-observations":             "There are no submitted review observations for this population.",
	"outside-history-read":                         "This pull request is outside the independent bounded history read.",
	"commit-list-changed":                          "The base and historical snapshots do not have matching totals and identical unique returned first-100 commit SHA sets.",
	"unavailable":                                  "The source evidence is unavailable.",
}
