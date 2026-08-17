package share

import (
	"fmt"
	"math"

	"github.com/assaio/assaio/internal/analyze"
	"github.com/assaio/assaio/internal/store"
)

// maxRays bounds what one card draws. Past it the picture stops gaining information and
// starts gaining bytes, so the excess is dropped -- and said out loud on the card, because
// a silently truncated picture reads as a complete one.
const maxRays = 3000

// PalSlots is how many colours the card draws with; the last is the shared "everything else"
// slot. Exported so a reader of Ray.Hue knows the range without reading the template.
const PalSlots = 5

// hues maps a model to its palette slot: the four heaviest models by tokens keep their own
// colour and everything else shares the last, matching the stacked model bar so the same
// model is the same colour on both frames.
func hues(byModel []analyze.ModelStat) map[string]int {
	out := make(map[string]int, len(byModel))
	for i := range byModel {
		slot := PalSlots - 1
		if i < PalSlots-1 {
			slot = i
		}
		out[byModel[i].Model] = slot
	}
	return out
}

// fingerprint draws one ray per session: angle from the hour it started, length from the
// output it produced, colour from the model it ran. It is the one frame nobody else can
// render, because it needs per-session timing that only a local log carries.
func fingerprint(sessions []store.SessionRow, byModel []analyze.ModelStat) Fingerprint {
	fp := Fingerprint{Total: len(sessions)}
	if len(sessions) == 0 {
		return fp
	}
	palette := hues(byModel)
	var peak int64
	for i := range sessions {
		if sessions[i].OutputTokens > peak {
			peak = sessions[i].OutputTokens
		}
	}
	if peak == 0 {
		peak = 1
	}
	step := 1
	if len(sessions) > maxRays {
		step = (len(sessions) + maxRays - 1) / maxRays
	}
	for i := 0; i < len(sessions); i += step {
		s := &sessions[i]
		if s.FirstTs.IsZero() {
			continue
		}
		// Local, because rhythm reads Local() and the card prints its off-hours share on the
		// same frame. Stored timestamps are UTC, so drawing UTC hours rotated the clock by the
		// machine's offset against the number captioned beside it -- 31% off-hours implied by
		// the angles against the 39% analyze published, on a +02:00 machine.
		start := s.FirstTs.Local()
		hour := float64(start.Hour()) + float64(start.Minute())/60
		hue, known := palette[s.Model]
		if !known {
			hue = PalSlots - 1
		}
		fp.Rays = append(fp.Rays, Ray{
			Angle: hour / 24,
			Len:   ratio(s.OutputTokens, peak),
			Hue:   hue,
		})
	}
	fp.Drawn = len(fp.Rays)
	fp.Omitted = fp.Total - fp.Drawn
	fp.Note = "angle = hour started · length = output, scaled against this window's busiest session · colour = model"
	if fp.Omitted > 0 {
		fp.Note += fmt.Sprintf(" · %d of %d sessions drawn", fp.Drawn, fp.Total)
	}
	return fp
}

// ratio softens the long tail so a single enormous session does not flatten every other ray
// into the hub. It is a fourth root, so a ray at half length is a sixteenth of the tokens --
// which is why the card's legend says "scaled against this window's busiest session" instead
// of implying the linear reading "length = output tokens" invites.
func ratio(v, peak int64) float64 {
	if v <= 0 {
		return 0
	}
	return math.Sqrt(math.Sqrt(float64(v) / float64(peak)))
}
