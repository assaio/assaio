# OpenSSF Best Practices badge

## Why

Scorecard alert #5 (`CIIBestPracticesID`, Low) scores 0 because no badge entry exists. It is the
only one of the five open Scorecard alerts that a maintainer action can actually close: the other
four are either self-resolving or structural to a single-maintainer project.

Registration needs a bestpractices.dev account linked to GitHub, so it cannot be done from the
harness. This file is the questionnaire, pre-answered from the repository, for one sitting.

## Steps

1. Sign in at <https://www.bestpractices.dev/> with the GitHub account that owns `assaio/assaio`.
2. **Add project**, repository URL `https://github.com/assaio/assaio`. Many criteria auto-detect
   from the repository; the table below is what to check or fill by hand.
3. Submit at passing level, then add the badge to `README.md`:

   ```markdown
   [![OpenSSF Best Practices](https://www.bestpractices.dev/projects/<ID>/badge)](https://www.bestpractices.dev/projects/<ID>)
   ```

4. Scorecard runs weekly (`.github/workflows/scorecard.yml`); alert #5 closes on the first run
   after the badge is live.

## Answers

Everything below is Met unless the Answer column says otherwise.

### Basics

| Criterion | Answer | Evidence |
|---|---|---|
| `description_good` | Met | Repository description + `README.md` |
| `interact` | Met | `SUPPORT.md`, GitHub Discussions |
| `contribution` | Met | <https://github.com/assaio/assaio/blob/main/CONTRIBUTING.md> |
| `contribution_requirements` | Met | Same file — hard rules, gate, commit conventions |
| `floss_license` / `floss_license_osi` | Met | Apache-2.0 |
| `license_location` | Met | <https://github.com/assaio/assaio/blob/main/LICENSE> |
| `documentation_basics` | Met | `README.md` (install, usage), `SECURITY.md`, `PRIVACY.md` |
| `documentation_interface` | Met | <https://assaio.dev/docs/reference> — generated, drift fails `make test` |
| `sites_https` | Met | `assaio.dev` and `github.com` are HTTPS only |
| `discussion` | Met | GitHub Discussions + Issues, both searchable |
| `english` | Met | Repo rule: everything in code and docs is English |
| `maintained` | Met | Active; `CHANGELOG.md` shows the cadence |

### Change control

| Criterion | Answer | Evidence |
|---|---|---|
| `repo_public` / `repo_track` / `repo_distributed` | Met | Public git repository |
| `repo_interim` | Met | Every change lands on `main` as its own PR, not only at release |
| `version_unique` / `version_semver` / `version_tags` | Met | SemVer tags, immutable via the `immutable-release-tags` ruleset |
| `release_notes` | Met | <https://github.com/assaio/assaio/blob/main/CHANGELOG.md> — Keep a Changelog, seven headings |
| `release_notes_vulns` | **N/A** | No CVE has been fixed yet. Justify: "no known vulnerabilities fixed to date; `CHANGELOG.md` has a **Security** heading reserved for them." |

### Reporting

| Criterion | Answer | Evidence |
|---|---|---|
| `report_process` / `report_tracker` | Met | `.github/ISSUE_TEMPLATE/` (bug, feature, connector) + `SUPPORT.md` |
| `report_archive` | Met | <https://github.com/assaio/assaio/issues> |
| `report_responses` / `enhancement_responses` | **Choose: N/A or Met** | Measured 2026-09-23: no issue, pull request or discussion from anyone outside the project since the repository went public (2026-07-17). The only issues, #40 and #50, were opened by the nightly fuzz workflow; both are closed. Justify: "No bug reports or enhancement requests from outside the project to date." Pick N/A if the form offers it; Met with that sentence claims nothing more. |
| `vulnerability_report_process` | Met | <https://github.com/assaio/assaio/blob/main/SECURITY.md> |
| `vulnerability_report_private` | Met | Private reporting is enabled; `SECURITY.md` names the advisories form and a contact form |
| `vulnerability_report_response` | **N/A** | No vulnerability report received yet. `SECURITY.md` commits to 3 business days. |

### Quality

| Criterion | Answer | Evidence |
|---|---|---|
| `build` / `build_common_tools` / `build_floss_tools` | Met | `make build`; Go toolchain and make, both FLOSS |
| `test` / `test_invocation` | Met | `make test` → `go test ./...`, documented in `CONTRIBUTING.md` |
| `test_continuous_integration` | Met | `.github/workflows/ci.yml` on every PR and push, with `-race` |
| `test_most` | Met | 86.0% of statements (`go test ./... -coverprofile`, 2026-09-23). Lowest package: `internal/parser` (shared scanner) 59.1%; every parser package is above 82%. |
| `test_policy` / `tests_are_added` / `tests_documented_added` | Met | `CONTRIBUTING.md` **Tested.** — table-driven, golden files from real captures, `FuzzParse` per parser |
| `warnings` / `warnings_fixed` / `warnings_strict` | Met | `.golangci.yml`; `make lint` is a required status check, so a warning cannot merge |

### Security

| Criterion | Answer | Evidence |
|---|---|---|
| `know_secure_design` / `know_common_errors` | **Assert yourself** | Self-assessment by the primary developer; the evidence a justification can cite: `docs/threat-model.md` (trust surfaces, data map, deletion test); offline by default, the network reached only by `sync` and `serve`; least-privilege workflow tokens (`permissions: contents: read`) and SHA-pinned actions; untrusted input (session logs, plugin output, CSV exports) parsed behind validation and fuzzed; HTML rendered through `html/template`; no SQL built by string formatting; bearer token compared in constant time; gosec, CodeQL and govulncheck in CI. |
| `crypto_published` / `crypto_floss` / `crypto_working` / `crypto_weaknesses` | Met | Go standard library only: HMAC-SHA256 (`internal/pseudonym`), `crypto/subtle` constant-time compare (`internal/server/identity.go`) |
| `crypto_keylength` | Met | SHA-256, 256-bit secrets |
| `crypto_call` | Met | No cryptography is reimplemented |
| `crypto_random` | Met | `crypto/rand` (`internal/pseudonym/pseudonym.go`) |
| `crypto_password_storage` | **N/A** | No passwords. The team server authenticates with a bearer token compared in constant time. |
| `crypto_pfs` | **N/A** | The server has no TLS of its own and is documented as belonging behind a reverse proxy (`docs/automation.md`, `docs/architecture.md`, `docs/threat-model.md`); this code terminates no TLS. |
| `delivery_mitm` / `delivery_unsigned` | Met | HTTPS/SSH only; tagged releases carry build provenance attestations (`gh attestation verify <artifact> -o assaio`) |
| `vulnerabilities_fixed_60_days` / `vulnerabilities_critical_fixed` | Met | `govulncheck` is a required status check (`vuln`); Dependabot security updates enabled |
| `no_leaked_credentials` | Met | Secret scanning **and** push protection enabled |

### Analysis

| Criterion | Answer | Evidence |
|---|---|---|
| `static_analysis` / `static_analysis_common_vulnerabilities` | Met | CodeQL (go, actions, python), `golangci-lint`, `govulncheck` |
| `static_analysis_fixed` | Met | Alerts are triaged; `vuln` gates the merge |
| `static_analysis_often` | Met | CodeQL on every push to `main` and every PR, plus weekly |
| `dynamic_analysis` / `dynamic_analysis_enable_assertions` | Met | Native Go fuzzing, 30s per target on a PR and 10m per target nightly; CI tests run with `-race` |
| `dynamic_analysis_unsafe` | **N/A** | Go is memory-safe; no package under `internal/` or `cmd/` imports `unsafe`. |
| `dynamic_analysis_fixed` | Met | A nightly finding opens an issue automatically (`.github/workflows/fuzz.yml`, `report` job) |

## Handoff

Nothing in the repository is blocking. `test_most` is measured (Met, 86.0%) and
`report_responses` has its facts (no outside reports yet; choose N/A or Met). What is left is the
maintainer's own: the two self-assessments (`know_secure_design`, `know_common_errors`) and the
registration itself. Everything else is answerable from the table above.

After the badge is live: paste the badge markdown into `README.md` and delete this file.
