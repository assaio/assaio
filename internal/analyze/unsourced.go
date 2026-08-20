package analyze

// A verdict needs a line, and a line needs an authority: a published definition, a target the
// user set, or one this window's own distribution derives. Several of this package's lines had
// none of the three -- a number picked once, rendered in colour, and read afterwards as
// knowledge (B177). Those metrics keep every figure and give up the verdict, which is the only
// half of them assaio was not entitled to.
//
// burn and edit_loops are what a sourced line looks like here: both derive theirs from the
// window's own median and MAD at the conventional 3.5 cutoff (see dispersion.go).

// reportedRead is the Read for a metric that reports a figure and does not judge it. The
// dashboard renders "neutral" without a favorable or unfavorable colour, and Rank never promotes
// it, so a number with no line behind it cannot lead a report either. One shared value rather
// than a per-metric word: the reader has to be able to see at a glance that these are the same
// kind of answer.
var reportedRead = Read{Key: "neutral", Label: "REPORTED"}

// unsourcedLine is the caveat every descriptive read carries. It names the missing authority
// rather than implying the figure is weak: the measurement is as good as it ever was, and what
// is gone is a good/bad call this project could not defend.
func unsourcedLine(quantity, whatWouldSettleIt string) string {
	return "No verdict on purpose: nothing published defines where " + quantity +
		" becomes a problem, and this window's own distribution cannot derive that line either, so assaio reports the figure and leaves the call to whoever knows the work. " +
		whatWouldSettleIt + " would settle it; until then a colour here would be assaio's opinion wearing a measurement's clothes."
}

// ownHistoryWouldSettleIt is the shared answer to "what would settle it" for a rate whose only
// credible baseline is the same store's earlier windows -- the comparison assaio cannot make
// until it keeps per-metric history (see BACKLOG.md).
const ownHistoryWouldSettleIt = "Your own earlier windows, once assaio keeps enough history to compare a rate against the one you used to run at,"
