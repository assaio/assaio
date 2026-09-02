package threshold

import (
	"errors"
	"strings"
)

// register holds every adjudicated candidate. Populated by candidates.go's init(); never
// mutated after program start.
var register []Candidate

// Register admits one adjudicated candidate. Every field is required and a missing one panics
// at program start rather than rendering as an empty string beside a figure: a citation with no
// URL, no population or no expiry is precisely the unfalsifiable authority this register exists
// to keep out, and shipping one must not be possible quietly. Every test binary in the module
// runs this init, so an incomplete entry is a red build.
func Register(c *Candidate) {
	if err := validate(c); err != nil {
		panic("threshold: " + err.Error())
	}
	register = append(register, *c)
}

// For returns the candidates weighed against an assaio metric, in registration order. A metric
// with none registered gets none: silence about a comparison nobody has made.
func For(metric string) []Candidate {
	var out []Candidate
	for i := range register {
		if register[i].Metric == metric {
			out = append(out, register[i])
		}
	}
	return out
}

// All returns every registered candidate, in registration order.
func All() []Candidate {
	out := make([]Candidate, len(register))
	copy(out, register)
	return out
}

// validate rejects an entry that could not be checked at its source or could not expire. The
// name in every message is the pairing, since the same citation is weighed against more than
// one metric and only the pairing identifies which row is wrong.
func validate(c *Candidate) error {
	name := c.Metric + "/" + c.Citation.Source
	for _, f := range []struct{ field, value string }{
		{"Metric", c.Metric},
		{"Differs", c.Differs},
		{"Citation.Value", c.Citation.Value},
		{"Citation.Definition", c.Citation.Definition},
		{"Citation.Source", c.Citation.Source},
		{"Citation.URL", c.Citation.URL},
		{"Citation.Population", c.Citation.Population},
	} {
		if strings.TrimSpace(f.value) == "" {
			return errors.New(name + " leaves " + f.field + " empty")
		}
	}
	if !strings.HasPrefix(c.Citation.URL, "https://") {
		return errors.New(name + " cites " + c.Citation.URL + ", which a reader cannot resolve")
	}
	switch {
	case c.Citation.Measured.IsZero():
		return errors.New(name + " states no measurement date")
	case c.Citation.Checked.IsZero():
		return errors.New(name + " states no date assaio last read the source")
	case c.Citation.Checked.Before(c.Citation.Measured):
		return errors.New(name + " was read before the measurement it quotes ends")
	case !c.Citation.ValidUntil.After(c.Citation.Measured):
		// An expiry at or before the measurement is a citation that was never in force, which
		// reads in a diff exactly like one that is.
		return errors.New(name + " expires no later than the measurement it quotes")
	}
	return validateFit(name, c.Fit)
}

// validateFit requires every defining property to be addressed exactly once. A property left
// out is the one nobody looked at, and Fits would then pass on a comparison that was never made.
func validateFit(name string, fit []Comparison) error {
	seen := map[Property]bool{}
	for i := range fit {
		f := &fit[i]
		if strings.TrimSpace(f.Cited) == "" || strings.TrimSpace(f.Assaio) == "" {
			return errors.New(name + " leaves the " + string(f.Property) + " comparison undescribed")
		}
		if seen[f.Property] {
			return errors.New(name + " compares " + string(f.Property) + " twice")
		}
		seen[f.Property] = true
	}
	for _, p := range Properties() {
		if !seen[p] {
			return errors.New(name + " never compares " + string(p))
		}
	}
	return nil
}
