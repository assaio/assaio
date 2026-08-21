package openmetrics

// Limits bound one parse. They are arguments rather than constants because the two adapters
// read very different documents -- a DCGM exporter on a dense node emits a sample per GPU per
// metric -- and because a limit nobody can see is a limit nobody can raise.
type Limits struct {
	// MaxBytes bounds the whole document.
	MaxBytes int64
	// MaxLineBytes bounds one line; a longer one fails the parse rather than being truncated
	// into a different sample.
	MaxLineBytes int
	// MaxFamilies bounds distinct metric names.
	MaxFamilies int
	// MaxSamplesPerFamily bounds label cardinality within one metric.
	MaxSamplesPerFamily int
	// MaxLabelsPerSample bounds labels on one sample.
	MaxLabelsPerSample int
	// MaxLabelValueBytes bounds one label value, which is the part an exporter can be made to
	// echo and which reaches a rendered report.
	MaxLabelValueBytes int
}

// DefaultLimits are sized for a real exporter with headroom, not for a hostile one: a DCGM
// node with 8 GPUs and every field enabled emits on the order of a thousand samples.
func DefaultLimits() Limits {
	return Limits{
		MaxBytes:            8 << 20,
		MaxLineBytes:        64 << 10,
		MaxFamilies:         2000,
		MaxSamplesPerFamily: 5000,
		MaxLabelsPerSample:  32,
		MaxLabelValueBytes:  512,
	}
}

// orDefaults fills any bound the caller left at zero, so a zero-value Limits is the default
// rather than a parser that reads nothing.
func (l Limits) orDefaults() Limits {
	d := DefaultLimits()
	if l.MaxBytes <= 0 {
		l.MaxBytes = d.MaxBytes
	}
	if l.MaxLineBytes <= 0 {
		l.MaxLineBytes = d.MaxLineBytes
	}
	if l.MaxFamilies <= 0 {
		l.MaxFamilies = d.MaxFamilies
	}
	if l.MaxSamplesPerFamily <= 0 {
		l.MaxSamplesPerFamily = d.MaxSamplesPerFamily
	}
	if l.MaxLabelsPerSample <= 0 {
		l.MaxLabelsPerSample = d.MaxLabelsPerSample
	}
	if l.MaxLabelValueBytes <= 0 {
		l.MaxLabelValueBytes = d.MaxLabelValueBytes
	}
	return l
}
