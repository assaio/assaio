package event

// The closed vocabularies every envelope draws on. They live in one file for the reason
// internal/label does: a collector, the validator and any future catalog must not be able to
// disagree about what a valid value is. Adding a value is a one-line change here.
//
// Privacy, provenance and time source are classifications every observation is stamped with,
// so they keep their full sets whether or not this build happens to emit each value: the
// meaning of local-only is the contrast with pseudonymous, and a one-value classification
// classifies nothing. Grain is not like them; the next block says why.

// The grains an observation can be stated at. A grain is the *unit a payload counts in*, so it
// lives and dies with its type: turn and session left with the AI payloads (ADR 0016), and a
// pull request or a check run adds its own with the collector that emits it. A vocabulary
// listing a unit nothing can produce invites a consumer to plan for evidence that never
// arrives.
const (
	GrainCommit = "commit"
)

// The privacy classes. They say what a payload *is*; what may be sent where is the
// correlation threat model's decision (B100), not this package's.
const (
	// LocalOnly must never leave the machine under any configuration.
	LocalOnly = "local-only"
	// Pseudonymous may be synced once identities are pseudonymized.
	Pseudonymous = "pseudonymous"
	// PublicMetadata says nothing about a person or a repository's contents.
	PublicMetadata = "public-metadata"
)

// How an observation came to exist. Deliberately no "estimated" or "attributed": those
// describe a signal (B90) or an attribution edge (B85), neither of which is an observation.
const (
	// Parsed was read from an artifact the source produced.
	Parsed = "parsed"
	// Derived was computed by assaio from other observations.
	Derived = "derived"
	// Manual was stated by a person.
	Manual = "manual"
)

// Where OccurredAt came from. Only the first is the source's own claim about when something
// happened; the other two are assaio guessing, and a consumer deserves to know which.
const (
	TimeStated     = "source-stated"
	TimeFileMtime  = "file-mtime"
	TimeIngestTime = "ingest-time"
)

// The vocabularies in declaration order, for validation and for the error messages that name
// what was expected. They stay unexported: what a consumer needs is the signal catalog
// (B90), not four accessors nothing calls yet.
var (
	grains      = []string{GrainCommit}
	privacies   = []string{LocalOnly, Pseudonymous, PublicMetadata}
	provenances = []string{Parsed, Derived, Manual}
	timeSources = []string{TimeStated, TimeFileMtime, TimeIngestTime}
)

func valid(vocabulary []string, v string) bool {
	for _, allowed := range vocabulary {
		if allowed == v {
			return true
		}
	}
	return false
}
