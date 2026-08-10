# Reconciliation samples

| file | capture | shape verified against |
|------|---------|------------------------|
| `constructed-export.csv` | **constructed** | nothing — column names and magnitudes are invented |

`capture: constructed` carries the same meaning as in `internal/calibration`: the sample
proves the reader and the arithmetic, not that a vendor writes this shape. No real export
has been read by this repository, which is why `internal/reconcile` binds columns by
header alias and prints what it bound instead of shipping a named vendor profile.

Replacing this with a redacted real export — from the Anthropic Console, the OpenAI usage
page, or any other vendor — is the one contribution that would upgrade this to `real`.
Redact to the allowlisted fields only: date, model, token count, amount, currency.
