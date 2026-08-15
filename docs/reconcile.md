# Reconciling against the vendor's own numbers

Every `$` assaio prints is an **estimate**: local token counts priced at public
pay-as-you-go rates. `assaio-agent reconcile` is the only check on that estimate which does
not come from assaio itself — it compares the estimate against what a vendor says it
billed.

```
assaio-agent reconcile ~/Downloads/usage.csv --since 30d
```

No credential and no network. The export is a file you downloaded; getting it out of a
vendor console is deliberately your step, so the tool never holds a key that could pull
your account data.

## Getting an export

Every vendor puts this somewhere different, and the location changes more often than this
file does. Look for "usage", "billing", "cost" or "activity" in the vendor's console and an
**Export** or **Download CSV** control on that page. Any export works as long as each row
carries a **date** and an **amount**; a model column and a token column make the report
say more, and their absence is reported rather than assumed away.

## Column binding

assaio has never read a real vendor export, so it ships **no vendor profiles** — claiming
"supports X" for a format nobody here has verified is exactly the kind of unchecked claim
this project refuses. Instead columns bind by header alias, and the binding is printed:

```
  read as     day=Usage Date  cost=Cost USD  model=Model  currency=Currency
```

Check that line. If a column bound wrong, or an alias missed, say so explicitly:

```
assaio-agent reconcile usage.csv --map cost=amount,day=usage_date,model=sku
```

Only `day` and `cost` are required. A non-USD export is refused rather than converted at a
rate assaio would have to invent.

An export over 64 MiB is refused rather than read — narrow the exported range and run it again.

## Reading the output

**Scope comes first, and it is not a formality.** An export's window is almost never the
window you queried, and comparing two totals over two different date ranges reports a
difference in *dates* as a difference in *money*. The overlap is computed before any delta,
and the money excluded on each side is printed so the exclusion is visible.

**The estimate may be a band.** Usage assaio read but could not price is never added into
the figure; it sets the top of a band instead, extrapolated at the window's own $/token.
That extrapolation is crude by construction — a cache read and a completion token do not
cost the same — which is why it sizes a range and never becomes a number of its own.

**Only evidenced causes are named.** A per-model cause is computed only when the two sides'
model names actually share values. When the vendor writes `claude-sonnet-4.5` and the logs
write `claude-sonnet-4-5-20250929`, matching them is guesswork, so the run says the
vocabularies have nothing in common instead of inventing an explanation.

**The residual is the point.** Whatever no named cause accounts for is printed as
*unexplained delta*. It is not rounded away, not absorbed into the nearest cause, and
neither side is adjusted to close it. A delta assaio cannot explain is the output.

## What it cannot check

Printed on every run, so silence is never mistaken for coverage:

- a line count, an edit attribution, or a per-session split — an export bills aggregates
- anything on a **flat-rate plan**: a subscription bills the plan, not the tokens, so there
  is no per-token actual to compare against
- whether the price table itself is right — this checks where totals land, not what a token
  should have cost

## Contributing a capture

`internal/reconcile/testdata/` holds a **constructed** sample, labelled as such in the same
way `internal/calibration` labels its traces. One redacted real export — date, model, token
count, amount, currency, nothing else — would upgrade that to `real` and let the aliases be
checked against a format a vendor actually writes.
