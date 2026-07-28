package i18n

// CLI is the terminal chrome: today only the statusline's fixed words, since every other
// command's output is either data or a validator's own text.
type CLI struct {
	StatuslinePrefix       string // leading marker, so the segment is identifiable in a shared status bar
	StatuslineNoData       string // shown when the store is missing or empty
	StatuslineNothingToday string // shown when the store has data but none for the local day
	StatuslineTokens       string // unit suffix for the token count
	StatuslineLines        string // unit suffix for AI-added lines
	StatuslineAPIEquiv     string // marks the $ figure as an API-equivalent estimate, not a bill
	StatuslineMonth        string // suffix on the month-to-date vs plan comparison
	AgeJustNow             string // ingest age under a minute
	AgeMinutes             string // ingest age in minutes, one %d
	AgeHours               string // ingest age in hours, one %d
	AgeDays                string // ingest age in days, one %d
	AgeUnknown             string // the store predates ingest tracking, so freshness is genuinely unknown
}

var enCLI = CLI{
	StatuslinePrefix:       "assaio",
	StatuslineNoData:       "no data (run backfill)",
	StatuslineNothingToday: "nothing today",
	StatuslineTokens:       "tok",
	StatuslineLines:        "lines",
	StatuslineAPIEquiv:     "API-equiv",
	StatuslineMonth:        "mo",
	AgeJustNow:             "just now",
	AgeMinutes:             "%d min ago",
	AgeHours:               "%dh ago",
	AgeDays:                "%dd ago",
	AgeUnknown:             "age unknown",
}
