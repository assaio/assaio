package reprice

import (
	"fmt"
	"strconv"
	"strings"
)

// Where a plan price came from. assaio vendors no plan catalogue on purpose: published plan
// prices change without notice, and a stale one read as current is a wrong recommendation
// about real money. The figure comes from whoever is looking at the vendor's page, and Source
// records which of them it was so a reader can go and check it.
const (
	SourceConfig = "config pricing.monthly_subscription_cost"
	SourceFlag   = "--plan, your figure from the vendor's page"
)

// Plan is one flat monthly price beside the window's projected monthly re-priced rate.
type Plan struct {
	Name string `json:"name"`
	// Monthly is the flat price in dollars, exactly as the caller stated it.
	Monthly float64 `json:"monthly"`
	Source  string  `json:"source"`
	// Multiple is the projected monthly API-equivalent over Monthly: above 1 the plan returns
	// more than its price at this volume, below 1 it does not. It is a ratio of an estimate to
	// a real price, never a saving -- the plan's own bill is the only figure that is neither.
	// Nil where the window has no priced usage: a zero there reads as a plan returning nothing,
	// which is a verdict on the plan drawn from the absence of evidence about it.
	Multiple *float64 `json:"multiple"`
}

// ParsePlan reads a "name=price" candidate, e.g. "Max 20x=200". The name is split at the last
// separator so a plan whose name contains one still parses.
func ParsePlan(s string) (Plan, error) {
	i := strings.LastIndex(s, "=")
	if i < 0 {
		return Plan{}, fmt.Errorf("plan %q must be name=monthly-price, e.g. \"Max 20x=200\"", s)
	}
	name := strings.TrimSpace(s[:i])
	if name == "" {
		return Plan{}, fmt.Errorf("plan %q has no name", s)
	}
	price, err := strconv.ParseFloat(strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(s[i+1:]), "$")), 64)
	if err != nil {
		return Plan{}, fmt.Errorf("plan %q: %q is not a monthly price", s, s[i+1:])
	}
	if price <= 0 {
		return Plan{}, fmt.Errorf("plan %q: a monthly price must be above zero, got %g", s, price)
	}
	return Plan{Name: name, Monthly: price, Source: SourceFlag}, nil
}

// plans puts the configured plan first and the caller's candidates after it, in the order they
// were given, and computes each multiple against the same projected monthly rate. A window with
// no priced usage leaves every multiple unset: the plan is still listed, because what it costs
// is a fact, but nothing here has a side to compare it against.
func plans(b *Basis, configured float64, supplied []Plan) []Plan {
	out := make([]Plan, 0, len(supplied)+1)
	if configured > 0 {
		out = append(out, Plan{Name: "configured plan", Monthly: configured, Source: SourceConfig})
	}
	out = append(out, supplied...)
	for i := range out {
		if !b.Priced {
			continue
		}
		m := share(b.Monthly, out[i].Monthly)
		out[i].Multiple = &m
	}
	return out
}
