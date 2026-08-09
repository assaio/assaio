// Package calibration holds the checks whose expected answer does not come from the parser
// that produces it.
//
// The honesty machinery this project already has answers "is this field present, and how
// much does it cover?". v0.12 answered yes to all of it while being twice wrong, because the
// field was present, complete, and counted at the wrong semantic grain -- and a golden file
// written from that reading preserved the mistake for eleven releases. A golden is a
// regression test for a parser against itself; nothing in it can notice that the parser was
// wrong on the day it was written.
//
// So every expectation here has a source outside internal/parser. Four kinds, and no test in
// this package may derive an expected value any other way:
//
//   - An adjudicated trace: a redacted real sample whose totals were counted from the raw
//     bytes, with each figure's derivation written down beside it (trace.go).
//   - An accounting invariant: a relation that must hold whatever the numbers are, because
//     the domain says so -- one logical response billed once, a subset never above its whole
//     (invariants.go).
//   - A metamorphic property: two encodings of the same fact that must produce the same
//     total, so neither encoding can be read correctly alone (metamorphic.go).
//   - A cross-surface assertion: report, status, the dashboard, sync and a metric plugin
//     rendering one window, which may not disagree (surface.go).
//
// Redaction is by field allowlist: a trace carries the numbers the tool wrote and nothing
// else, so no path, prompt or file body reaches this repository.
package calibration
