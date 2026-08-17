package share

import (
	"strconv"
	"strings"

	"github.com/assaio/assaio/internal/analyze"
)

// verdicts indexes one window's analyze output by validator name, so every read below
// names the metric it is quoting rather than re-deriving it.
type verdicts map[string]*analyze.Result

func index(results []analyze.Result) verdicts {
	out := make(verdicts, len(results))
	for i := range results {
		out[results[i].Name] = &results[i]
	}
	return out
}

// figure returns a validator's figure by its published label. The empty string means the
// validator did not run or does not publish that label in this window -- never a zero,
// which a caller must not read as one.
func (v verdicts) figure(validator, label string) (analyze.Figure, bool) {
	r, ok := v[validator]
	if !ok {
		return analyze.Figure{}, false
	}
	for _, f := range r.Figures {
		if f.Label == label {
			return f, true
		}
	}
	return analyze.Figure{}, false
}

func (v verdicts) value(validator, label string) string {
	f, _ := v.figure(validator, label)
	return f.Value
}

// share reads a figure whose published form is a percentage ("92.6%", ">99%", "5%") and
// returns it as 0..100. A bounded form (">99%") is read at its bound, which is the number
// the validator itself chose to publish rather than a guess past it. ok is false when the
// figure is absent or does not parse, and a caller then omits whatever rested on it.
func (v verdicts) share(validator, label string) (float64, bool) {
	f, ok := v.figure(validator, label)
	if !ok {
		return 0, false
	}
	s := strings.TrimSpace(f.Value)
	s = strings.TrimSuffix(s, "%")
	s = strings.TrimPrefix(s, ">")
	s = strings.TrimPrefix(s, "<")
	s = strings.TrimPrefix(s, "~")
	n, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return 0, false
	}
	return n, true
}

// count reads a figure published as a grouped integer ("2,521", "155,041"). Bounded and
// abbreviated forms ("41.1B") deliberately do not parse here: a hook that needs a real
// count reads it from the prepared aggregates instead of un-abbreviating a display string.
func (v verdicts) count(validator, label string) (int64, bool) {
	f, ok := v.figure(validator, label)
	if !ok {
		return 0, false
	}
	n, err := strconv.ParseInt(strings.ReplaceAll(strings.TrimSpace(f.Value), ",", ""), 10, 64)
	if err != nil {
		return 0, false
	}
	return n, true
}
