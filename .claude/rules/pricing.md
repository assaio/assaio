---
paths:
  - "internal/pricing/**"
  - "internal/reprice/**"
  - "internal/reconcile/**"
---

# Prices

- **Every `$` is a token count times `internal/pricing/litellm.json`**, or `retained.json` for a
  model it dropped. Refreshing it is
  `make prices`, a release step, not a chore: five weeks of drift once left 45.5% of tokens
  unpriced and a window $15,452.42 short (`RELEASING.md`, "The public surface").
- **An unprefixed key never leaves the table.** LiteLLM drops a model past its deprecation date;
  `retained.json` keeps it at its last price, and `TestRetainedHoldsEveryUnprefixedKey` fails a
  refresh that skipped the fold. Deleting a line of `retained.json` unprices that model's history.
  An entry listed with no token rate (zero or absent) is unpriced, never $0, and the ledger does
  not refill a key the snapshot still lists.
- **A model the table cannot cost widens the unpriced share**; it never rounds to zero.
  `TestEveryCalibratedModelHasAPrice` and `doctor --strict` catch what they can see, and neither
  sees a model the vendor shipped that nobody here has run.
- **Costs are API-equivalent estimates**, never an invoice or quota consumption; `reconcile` keeps
  the unexplained delta visible instead of adjusting anything.
- The snapshot is MIT-licensed; `NOTICE` carries the attribution.
