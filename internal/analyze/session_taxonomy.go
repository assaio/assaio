package analyze

import "strconv"

const (
	taxonomyName      = "session-taxonomy"
	taxonomyTitle     = "Session Taxonomy"
	taxonomyDescribe  = "What kind of work sessions are: conversational (no edits), light-edit, or heavy-edit, and how the mix splits."
	taxonomyHowToRead = "Conversational sessions are real work -- design, debugging, and planning rarely touch files -- so a high conversational share is not waste. This is a mix, not a scorecard: read it to understand how you use AI, not to push every session toward edits."
	// taxonomyLightMax is the top of the light-edit bucket (inclusive); above it is heavy.
	taxonomyLightMax = 10
	// taxonomyMinSessions is the floor below which the split is too thin to characterize.
	taxonomyMinSessions = 5
)

func init() { Register(taxonomyValidator{}) }

// taxonomyValidator buckets sessions by how much they edited: conversational (no edits),
// light-edit, or heavy-edit -- a descriptive map of how AI is used, never a scorecard.
type taxonomyValidator struct{}

func (taxonomyValidator) Name() string     { return taxonomyName }
func (taxonomyValidator) Title() string    { return taxonomyTitle }
func (taxonomyValidator) Describe() string { return taxonomyDescribe }

//nolint:gocritic // Input is required by the Validator interface; analyzed once per run, not a hot path.
func (taxonomyValidator) Analyze(in Input) Result {
	r := Result{Name: taxonomyName, Title: taxonomyTitle, Describe: taxonomyDescribe, HowToRead: taxonomyHowToRead}
	if len(in.Sessions) == 0 {
		r.Read = noDataRead
		r.Takeaway = "No sessions in this window."
		return r
	}
	var conv, light, heavy int64
	for i := range in.Sessions {
		switch e := in.Sessions[i].Edits; {
		case e == 0:
			conv++
		case e <= taxonomyLightMax:
			light++
		default:
			heavy++
		}
	}
	total := int64(len(in.Sessions))
	editing := light + heavy
	enough := total >= taxonomyMinSessions

	if enough {
		r.Read = readFor(true, taxonomyLabel(conv, editing, total))
	} else {
		r.Read = noDataRead
	}
	r.Purity = fracOf(editing, total)
	r.Figures = []Figure{
		{Label: "conversational", Value: shareOrDash(conv, total, 0), Note: "no file edits"},
		{Label: "light-edit", Value: shareOrDash(light, total, 0), Note: "1-" + strconv.Itoa(taxonomyLightMax) + " edits"},
		{Label: "heavy-edit", Value: shareOrDash(heavy, total, 0), Note: ">" + strconv.Itoa(taxonomyLightMax) + " edits"},
	}
	r.Bars = []Bar{
		taxonomyBar("conversational", conv, total),
		taxonomyBar("light-edit", light, total),
		taxonomyBar("heavy-edit", heavy, total),
	}
	r.Takeaway = taxonomyTakeaway(conv, editing, total, enough)
	r.Caveats = []string{
		"Conversational sessions are real work (design, debugging, planning) -- not idle time.",
		"A thrash bucket needs per-session rework, which isn't stored yet, so it isn't split out here.",
	}
	return r
}

func taxonomyBar(label string, n, total int64) Bar {
	return Bar{Label: label, Value: strconv.FormatInt(n, 10) + " (" + shareOrDash(n, total, 0) + ")", Frac: fracOf(n, total)}
}

// taxonomyLabel names the dominant bucket for the faceplate, short enough for the label slot.
func taxonomyLabel(conv, editing, total int64) string {
	switch {
	case conv*2 > total:
		return "Chat-led"
	case editing*2 > total:
		return "Edit-led"
	default:
		return "Mixed"
	}
}

func taxonomyTakeaway(conv, editing, total int64, enough bool) string {
	switch {
	case !enough:
		return "Too few sessions this window to characterize the mix."
	case conv*2 > total:
		return "Most sessions are conversational -- design and debugging work that rarely edits files."
	case editing*2 > total:
		return "Most sessions produce edits -- AI is doing hands-on building here."
	default:
		return "A balanced mix of conversational and edit-producing sessions."
	}
}
