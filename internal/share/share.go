// Package share builds the render-ready shape of an assay fit to post in public: the
// reel's frames, the still card, and the post text beside them.
//
// Nothing here originates a figure. Every number is read from what internal/analyze
// already published for the same window, so a number on a shared card cannot disagree
// with the same number in `assaio analyze` -- the one property that lets a marketing
// surface sit on top of a measurement tool without deforming it.
//
// Redaction is structural rather than a flag (B149): no field below can hold a
// repository, member, path, branch, skill or sub-agent name, so there is no code path
// that renders one. Tools and models are named, because which coding agent and which
// model ran is a public fact about a vendor, not about a person.
package share

// Assay is one window, shaped for the six frames the reel plays and the still card that
// leads it.
type Assay struct {
	// Window is the human label ("last 30 days"); Days is how many of them carried usage.
	Window string
	Days   int
	// Layer names the measurement layers the card's figures rest on.
	Layer string
	// Limit is the one line every rendered frame carries. It exists because Fine lives on the
	// preview page while the PNG and the MP4 are what actually leave the machine: a figure
	// whose limits stay behind travels as an unqualified fact.
	Limit string

	Hook      Hook
	Scale     []Stat
	Models    []Slice
	Tools     []Slice
	Archetype Archetype
	Rays      Fingerprint
	Profile   Profile
	Ledger    []Stat

	// Post is the ready-to-paste text; Fine is the card's own small print, every line of
	// which is a claim limit rather than a decoration.
	Post string
	Fine []string

	// Hashtag, Install and ShareCmd are what turn a viewer into the next poster, and they
	// are on the artwork rather than only in the caption: a caption is dropped when an image
	// is re-shared, and the whole point of the image is that it travels on its own.
	Hashtag  string
	Install  string
	ShareCmd string
}

// Stat is one labelled number as a validator already formatted it.
type Stat struct {
	Label string
	Value string
	Note  string
}

// Slice is one band of a stacked proportion: a share of a whole, with the palette index
// the renderer paints it in.
type Slice struct {
	Label string
	Value string
	// Frac is 0..1 of the whole this band covers.
	Frac float64
	// Hue is 0..4 into the card palette; 4 is the "everything else" grey.
	Hue int
}

// Hook is the first frame -- the one line that decides whether anyone watches the rest.
// Universal marks the three that carry no threshold and therefore always resolve, which
// is what makes an empty first frame unreachable.
type Hook struct {
	Family    string
	Line      string
	Sub       string
	Universal bool
}

// Archetype is the shape of how someone works, never a rank: Index of Of, with a flavor
// line that quotes the figures that chose it. Deliberately carries no percentile -- that
// needs a population, and a population does not exist until people run this.
type Archetype struct {
	Name   string
	Flavor string
	Index  int
	Of     int
}

// Ray is one session in the fingerprint: Angle is its start hour on a 24-hour clock
// (0..1 of a turn), Len is 0..1 of the longest session's output, Hue indexes the same
// palette as Slice, from the session's own model.
type Ray struct {
	Angle float64
	Len   float64
	Hue   int
}

// Fingerprint is the drawn sessions and, when the window holds more than the card can
// carry, how many were left out. A cap is stated rather than applied silently: a picture
// that quietly dropped half its input would read as a complete one.
type Fingerprint struct {
	Rays    []Ray
	Drawn   int
	Total   int
	Note    string
	Omitted int
}

// Axis is one dimension of the practice profile. Both ends are legitimate positions, so
// there is no better end and no grade; At is 0..1 from Left to Right. Exactly one Axis in
// a Profile carries Reserve.
type Axis struct {
	Left    string
	Right   string
	Value   string
	At      float64
	Reserve bool
}

// Profile is the five axes and the single reserve drawn from them. Every card has exactly
// one reserve by construction, which is what stops the weak half reading as a verdict on
// the person holding it.
type Profile struct {
	Axes    []Axis
	Title   string
	Line    string
	Caveats []string
}
