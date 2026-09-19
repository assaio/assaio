---
paths:
  - "internal/pricing/**"
  - "internal/reprice/**"
  - "internal/reconcile/**"
---

# Prices

- **Every `$` is a token count times `internal/pricing/litellm.json`.** Refreshing it (and
  `SnapshotDate` in `snapshot.go`) is a release step, not a chore: five weeks of drift once left
  45.5% of tokens unpriced and a window $15,452.42 short (`RELEASING.md`, "The public surface").
- **A model the table cannot cost widens the unpriced share**; it never rounds to zero.
  `TestEveryCalibratedModelHasAPrice` and `doctor --strict` catch what they can see, and neither
  sees a model the vendor shipped that nobody here has run.
- **Costs are API-equivalent estimates**, never an invoice or quota consumption; `reconcile` keeps
  the unexplained delta visible instead of adjusting anything.
- The snapshot is MIT-licensed; `NOTICE` carries the attribution.
