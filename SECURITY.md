# Security Policy

## Reporting a vulnerability

Do not open a public GitHub issue for security vulnerabilities.

Report privately via [GitHub Security Advisories](https://github.com/assaio/assaio/security/advisories/new) (preferred) or through the contact form at [karauda.com/contact](https://karauda.com/contact), with a description of the issue, steps to reproduce, and
any relevant logs or proof of concept. Encrypt with PGP if the report is sensitive and
you'd like to request a key first.

We aim to acknowledge new reports within **3 business days** and to provide a fix or
mitigation plan within **30 days**, depending on severity. We'll credit you in the
release notes unless you ask to stay anonymous.

## Supported versions

assaio is pre-1.0. Security fixes land on the latest released minor version only.
There is no long-term support branch yet; this policy will be revisited at 1.0.

| Version   | Supported |
|-----------|-----------|
| `main`    | Yes       |
| Latest tag| Yes       |
| Older tags| No        |

## Scope

In scope: the `assaio-agent` CLI and the parsers/analytics shipped in this repository.
Out of scope: third-party AI coding tools whose data assaio ingests.

## Supply chain

CI Actions are pinned by commit SHA, Dependabot keeps Go modules and Actions current,
and `govulncheck` runs on every PR. Tagged releases carry GitHub build provenance
attestations for their artifacts; verify with
`gh attestation verify <artifact> -o assaio`.
