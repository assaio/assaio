package config

// DefaultTraceHorizonDays is how far back the step timeline is kept. Thirty days because that
// is what the sources themselves keep: Claude Code deletes transcripts after 30 by default, so
// a longer horizon holds history no re-read could ever rebuild. It is also a size bound the
// table needs -- the timeline and its indexes occupy roughly twice what the usage table does.
const DefaultTraceHorizonDays = 30

// Trace bounds how much of the session-step sequence the store keeps. The timeline is the
// largest thing assaio writes per unit of usage, so it is the one table with a horizon rather
// than an open-ended history.
type Trace struct {
	// HorizonDays is the age past which steps are pruned on the next ingest. An absent key
	// takes DefaultTraceHorizonDays from Defaults(); an explicit 0 means keep everything and
	// is honoured rather than coerced, because 0 is what a person writes to mean "no horizon"
	// and silently pruning to 30 days would delete history they asked to keep. Negative is
	// rejected by Validate. Widening it later does not bring pruned steps back on its own:
	// ingest skips transcripts it has already read, so recovering them takes `backfill --full`,
	// and only while the tool still has the files.
	HorizonDays int `koanf:"horizon_days"`
}
