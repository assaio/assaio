---
paths:
  - "internal/analyze/**"
  - "internal/dashboard/**"
  - "internal/report/**"
  - "internal/signal/**"
  - "internal/recommend/**"
  - "internal/drift/**"
  - "internal/threshold/**"
  - "internal/digest/**"
---

# Metrics, figures and the surfaces that render them

- **One metric = one file** in `internal/analyze/`, self-registered from `init()`; it appears in
  `analyze`, the dashboard and `docs/reference.json` automatically (`docs/extending/metric-validator.md`).
- **Every figure states its layer** — activity / output / outcome / impact (`internal/layer`,
  ADR 0013). Lines are output; presenting them as "did AI help" is how this product starts lying.
- **Absence is `—`, never `0`**: `PercentOrDash`, a widened unpriced share, an explicit
  unexplained delta. A divide-by-zero that prints `0%` or `100%` is a defect.
- **The denominator declares its scope** (`internal/trace`, `analyze.TraceReader`): a detector
  over sequences names one scope and renders the excluded share beside its figure.
- **A threshold has a source** (`internal/threshold`, with expiry) or is derived from the window's
  own median and spread. A number somebody picked is not allowed to look precise.
- **A pattern is not a fault**: every detector documents what its pattern cannot be told apart from.
- **Provenance and confidence travel with the fact** (`analyze.Result`); session-level and daily
  vendor aggregates are never blended untagged. Prices come only from `internal/pricing`.
- **Never a per-person leaderboard**: `BarsPseudonym`, pseudonymized by default at every export
  boundary (`internal/pseudonym`); the refusals at the bottom of `BACKLOG.md` hold.
- **Dashboard golden files** (`internal/dashboard/testdata/*.golden.html`) change whenever a
  validator's text changes; regenerate with `-update` and read the diff, then run `/ui-check`.
- A measurement change is not done until `corpus-prover` showed the figure that should move
  moved on the real corpus and nothing else did.
