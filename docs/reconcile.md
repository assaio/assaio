# Reconciling against the vendor's own numbers

Every `$` assaio prints is an **estimate** based on local token counts and public pay-as-you-go
rates. `assaio-agent reconcile` checks that estimate against a vendor's billed amount, the only
check from outside assaio.

```
assaio-agent reconcile ~/Downloads/usage.csv --since 30d
```

No credentials or network access are needed. You download the export from the vendor console, so the
tool never holds a key that can access your account data.

## Getting an export

Vendor console locations vary and change often. Look for "usage", "billing", "cost", or "activity",
then an **Export** or **Download CSV** control. Any export works if every row has a **date** and
**amount**. Model and token columns add detail; the report notes when they are missing.

## Column binding

assaio has never read a real vendor export, so it has **no vendor profiles** and makes no unverified
format support claims. It binds columns by header alias and prints the binding:

```
  read as     day=Usage Date  cost=Cost USD  model=Model  currency=Currency
```

Check that line. If a column bound incorrectly or an alias was missed, specify the binding:

```
assaio-agent reconcile usage.csv --map cost=amount,day=usage_date,model=sku
```

Only `day` and `cost` are required. Non-USD exports are rejected; assaio does not invent a
conversion rate.

Exports over 64 MiB are rejected. Narrow the date range and retry.

## Reading the output

**Compare scope first.** An export's date window rarely matches the query window. Comparing
different windows confuses a date difference with a money difference. The tool computes the overlap
before any delta and prints the amount excluded from each side.

**The estimate may be a band.** Usage assaio reads but cannot price is excluded from the estimate.
Instead, it sets the top of a band by extrapolating the window's own $/token. This is crude because
cache reads and completion tokens cost different amounts, so the extrapolation defines a range and
never appears as its own figure.

**Only evidenced causes are named.** The tool calculates a per-model cause only when both sides use
some identical model names. If the vendor says `claude-sonnet-4.5` and logs say
`claude-sonnet-4-5-20250929`, it reports that the names have no overlap instead of guessing a match.

**The residual is the point.** The tool prints any delta left after named causes as *unexplained
delta*. It neither rounds it away, assigns it to another cause, nor adjusts either side to erase it.
An unexplained delta is a valid result.

## What it cannot check

Printed on every run so missing coverage is clear:

- line counts, edit attribution, or per-session splits; exports bill aggregates
- anything on a **flat-rate plan**; subscriptions bill for the plan, not per token, so there is no
  per-token actual to compare
- whether the price table is correct; this checks where totals land, not what each token should cost

## Contributing a capture

`internal/reconcile/testdata/` contains a **constructed** sample, labeled like traces in
`internal/calibration`. One redacted real export containing only date, model, token count, amount,
and currency would change its label to `real` and let aliases be checked against a vendor's actual
format.
