package share

import (
	"fmt"

	"github.com/assaio/assaio/internal/humanize"
)

// axisSpec is one dimension of the practice profile. Both poles are legitimate positions
// -- there is no better end -- which is why this renders no letter and no score. reserve
// is how far this axis sits from its own comfortable middle, and only the widest one is
// marked; every card marks exactly one, so a marked axis is a rubric rather than a fault.
type axisSpec struct {
	left, right string
	at          func(f *facts) num
	value       func(f *facts) string
	reserve     func(f *facts) (title, line string, ok bool)
}

var axes = []axisSpec{
	{
		left: "thrifty", right: "premium",
		at:    func(f *facts) num { return f.premium },
		value: func(f *facts) string { return f.premium.text() + " premium tokens" },
		// The reserve is stated in the unit it was measured in -- the share of tokens, not
		// of cost. Reading a token share as a cost share is a scope swap that would let the
		// card publish a saving nothing here computed.
		reserve: func(f *facts) (string, string, bool) {
			if !f.premium.ok || f.premium.v < 60 {
				return "", "", false
			}
			return "model choice", f.premium.text() + " of my tokens ran on premium models", true
		},
	},
	{
		left: "talks", right: "edits",
		at:    func(f *facts) num { return invert(f.conversational) },
		value: func(f *facts) string { return f.conversational.text() + " of sessions without an edit" },
		reserve: func(f *facts) (string, string, bool) {
			if !f.conversational.ok || f.conversational.v < 92 {
				return "", "", false
			}
			return "sessions that produce", f.conversational.text() + " of sessions ended without touching a file", true
		},
	},
	{
		left: "solo", right: "delegates",
		at:    func(f *facts) num { return f.delegation },
		value: func(f *facts) string { return f.delegation.text() + " of work in sub-agents" },
		reserve: func(f *facts) (string, string, bool) {
			if !f.delegation.ok || f.delegation.v > 5 {
				return "", "", false
			}
			return "delegation", "almost nothing was handed to a sub-agent", true
		},
	},
	{
		left: "day", right: "night",
		at:    func(f *facts) num { return f.offHours },
		value: func(f *facts) string { return f.offHours.text() + " outside working hours" },
		reserve: func(f *facts) (string, string, bool) {
			if !f.offHours.ok || f.offHours.v < 65 {
				return "", "", false
			}
			return "when the work happens", f.offHours.text() + " of it ran outside working hours", true
		},
	},
	{
		left: "first try", right: "reworks",
		at:    func(f *facts) num { return scale(f.rework, 4) },
		value: func(f *facts) string { return f.rework.text() + " of lines rewritten" },
		reserve: func(f *facts) (string, string, bool) {
			if !f.rework.ok || f.rework.v < 15 {
				return "", "", false
			}
			return "rework", f.rework.text() + " of written lines came back out again", true
		},
	},
}

// buildProfile renders the five axes and marks the single reserve. When no axis clears
// its own condition the reserve falls back to the widest deviation from centre, so the
// rubric is never empty -- an empty one would read as "nothing to improve", which is a
// claim no window supports.
func buildProfile(f *facts) Profile {
	p := Profile{}
	widest, widestAt := -1, 0.0
	for i := range axes {
		got := axes[i].at(f)
		if !got.ok {
			continue
		}
		at := clamp(got.v / 100)
		p.Axes = append(p.Axes, Axis{Left: axes[i].left, Right: axes[i].right, Value: axes[i].value(f), At: at})
		if d := deviation(at); d > widestAt {
			widest, widestAt = len(p.Axes)-1, d
		}
		if title, line, ok := axes[i].reserve(f); ok && p.Title == "" {
			p.Title, p.Line = title, line
			p.Axes[len(p.Axes)-1].Reserve = true
		}
	}
	if p.Title == "" && widest >= 0 {
		ax := &p.Axes[widest]
		ax.Reserve = true
		p.Title = "widest lean"
		// The pole is chosen from which side of centre the marker actually sits on. Naming
		// Right unconditionally published the opposite of the measurement: a window with 4%
		// rework leans to "first try" and was captioned "reworks".
		pole := ax.Left
		if ax.At > 0.5 {
			pole = ax.Right
		}
		p.Line = fmt.Sprintf("%s — %s", pole, ax.Value)
	}
	if len(p.Axes) > 0 {
		p.Caveats = append(p.Caveats, "Both ends of every axis are legitimate positions. This is a shape, not a score, and never an input to evaluating a person.")
	}
	if p.Title == "model choice" {
		p.Caveats = append(p.Caveats, "A premium share is a token share, not a bill: what any of it would have cost on a cheaper model depends on work only you can judge. Directional, not a recommendation.")
	}
	return p
}

func invert(n num) num {
	if !n.ok {
		return n
	}
	return num{v: 100 - n.v, ok: true}
}

// scale stretches a share that lives in a narrow real range (rework rarely passes 25%)
// across the axis, so the marker is readable rather than pinned to one end.
func scale(n num, by float64) num {
	if !n.ok {
		return n
	}
	return num{v: n.v * by, ok: true}
}

func clamp(v float64) float64 {
	switch {
	case v < 0.02:
		return 0.02
	case v > 0.98:
		return 0.98
	default:
		return v
	}
}

func deviation(at float64) float64 {
	if at > 0.5 {
		return at - 0.5
	}
	return 0.5 - at
}

func thousandsOf(n int64) string { return humanize.Int(n) }
