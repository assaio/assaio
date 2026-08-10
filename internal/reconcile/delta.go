package reconcile

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/assaio/assaio/internal/humanize"
	"github.com/assaio/assaio/internal/report"
)

// Cause is one named, evidenced part of the delta. Amount is signed in the direction that
// explains it: positive means it accounts for the export billing more than the estimate.
type Cause struct {
	Name     string  `json:"name"`
	Amount   float64 `json:"amount"`
	Evidence string  `json:"evidence"`
}

// Result is a whole reconciliation. Both totals are the raw ones; nothing here was adjusted
// to bring them together.
type Result struct {
	Source   string   `json:"source"`
	Binding  Binding  `json:"binding"`
	Currency string   `json:"currency"`
	Rows     int      `json:"rows"`
	Skipped  int      `json:"skipped"`
	Scope    Scope    `json:"scope"`
	Estimate Estimate `json:"estimate"`
	// ExportCost is the vendor's own money over the overlap.
	ExportCost float64 `json:"export_cost"`
	// Delta is ExportCost minus the priced estimate: positive means the vendor billed more
	// than the local logs account for.
	Delta  float64 `json:"delta"`
	Causes []Cause `json:"causes"`
	// Unexplained is what no named cause accounts for. It is the point of the whole
	// command and is never rounded away, absorbed into a cause, or reported as zero
	// because it was small.
	Unexplained float64  `json:"unexplained"`
	Limits      []string `json:"limits"`
	Refusals    []string `json:"refusals"`
}

// InBand reports whether the vendor's total lands inside the range the estimate could
// honestly claim, given usage assaio read but could not price.
func (r *Result) InBand() bool {
	return r.ExportCost >= r.Estimate.Priced && r.ExportCost <= r.Estimate.Upper()
}

// refusals are fixed: they are what a billing export cannot answer no matter how good the
// import is, and they are printed every run so no reader infers otherwise from silence.
var refusals = []string{
	"A line count, an edit attribution, or a per-session split. An export bills aggregates; none of those are in it.",
	"Anything on a flat-rate plan. A subscription bills the plan, not the tokens, so there is no per-token actual to compare against.",
	"Whether the price table is right. This checks where the totals land, not what a token should have cost.",
}

// Reconcile compares an export against the local estimate over the window [start, end].
// Scope is resolved first and every figure after it is restricted to the overlap.
func Reconcile(exp *Export, rows []report.Row, start, end time.Time) (*Result, error) {
	if exp.Currency != "" && exp.Currency != "USD" {
		return nil, fmt.Errorf("export is in %s; assaio prices in USD and will not invent a conversion rate", exp.Currency)
	}
	scope := buildScope(exp, estimateDays(rows), start, end)
	est := buildEstimate(rows, &scope)
	res := &Result{
		Source: exp.Source, Binding: exp.Binding, Currency: exp.Currency,
		Rows: exp.Total, Skipped: exp.Skipped,
		Scope: scope, Estimate: est,
		ExportCost: exp.Cost(scope.overlap),
		Refusals:   refusals,
	}
	res.Delta = res.ExportCost - est.Priced
	res.Causes, res.Limits = explain(exp, &est, &scope)
	res.Limits = append(res.Limits, importLimits(exp)...)

	var named float64
	for _, c := range res.Causes {
		named += c.Amount
	}
	res.Unexplained = res.Delta - named
	return res, nil
}

// explain names the parts of the delta that have evidence behind them. A cause with no
// evidence is not produced at all -- it becomes a limit saying why it could not be computed.
func explain(exp *Export, est *Estimate, scope *Scope) (causes []Cause, limits []string) {
	if est.Extrapolated > 0 {
		causes = append(causes, Cause{
			Name:     "usage assaio could not price",
			Amount:   est.Extrapolated,
			Evidence: humanize.Count(est.UnpricedTokens) + " tokens with no known model price, at this window's own $/token",
		})
	}
	if exp.Binding.Model == "" {
		return causes, append(limits, "The export has no model column, so neither per-model cause could be computed. Pass --map model=<column> if it has one under another name.")
	}
	exportByModel := map[string]float64{}
	for _, r := range exp.Rows {
		if scope.Overlap(r.Day) && r.Model != "" {
			exportByModel[strings.ToLower(strings.TrimSpace(r.Model))] += r.Cost
		}
	}
	localByModel := map[string]float64{}
	for m, c := range est.ByModel {
		localByModel[strings.ToLower(strings.TrimSpace(m))] += c
	}
	if !shareAnyModel(exportByModel, localByModel) {
		return causes, append(limits, fmt.Sprintf(
			"The export's model names and the local ones have no value in common (%s vs %s), so per-model causes cannot be computed. Matching them by guesswork would invent an explanation.",
			sample(exportByModel), sample(localByModel),
		))
	}
	if only := sumMissing(exportByModel, localByModel); only > 0 {
		causes = append(causes, Cause{
			Name:     "billed for models the local logs never recorded",
			Amount:   only,
			Evidence: "usage on this machine's account that assaio did not read: another machine, another tool, or the vendor's web UI",
		})
	}
	if only := sumMissing(localByModel, exportByModel); only > 0 {
		causes = append(causes, Cause{
			Name:     "local usage the export does not cover",
			Amount:   -only,
			Evidence: "models assaio priced that this export does not bill, e.g. a second vendor's tool in the same window",
		})
	}
	return causes, limits
}

// importLimits states what the import itself could not do, so a thin comparison is never
// mistaken for a clean one.
func importLimits(exp *Export) []string {
	var out []string
	if exp.Skipped > 0 {
		out = append(out, fmt.Sprintf("%d of %d export rows could not be read and were skipped, so the vendor total below is short by whatever they held.", exp.Skipped, exp.Total))
	}
	if exp.Currency == "" {
		out = append(out, "The export states no currency. It is read as USD; if it is not, every figure here is wrong by the exchange rate.")
	}
	if exp.Binding.Tokens == "" {
		out = append(out, "The export has no token column, so only money is compared, never token counts.")
	}
	return out
}

func shareAnyModel(a, b map[string]float64) bool {
	for k := range a {
		if _, ok := b[k]; ok {
			return true
		}
	}
	return false
}

// sumMissing totals the values in a whose key is absent from b.
func sumMissing(a, b map[string]float64) float64 {
	var sum float64
	for k, v := range a {
		if _, ok := b[k]; !ok {
			sum += v
		}
	}
	return sum
}

// sample names up to two keys so a reader can see the shape of each vocabulary.
func sample(m map[string]float64) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	if len(keys) == 0 {
		return "none"
	}
	sort.Strings(keys)
	if len(keys) > 2 {
		keys = keys[:2]
	}
	return strings.Join(keys, ", ")
}
