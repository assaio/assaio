---
paths:
  - "internal/cli/**"
  - "cmd/**"
  - "internal/i18n/**"
  - "internal/config/**"
---

# Commands, flags and user-facing text

- **One command = one file** in `internal/cli/`; `--help` is the contract and `docs/reference.json`
  is generated from it — `make docs` after any flag or command change, or `make test` goes red.
- **A new top-level command must be named on `site/index.html`** (`data-covers="commands"`) and in
  the README command list; `internal/docs` fails the build in both directions.
- **Help text and i18n strings are product content**: English, one fact one place, and every
  figure's caveat travels with it. Text a reader sees is reviewed through the GPT/Gemini door
  (`/content-model`), not rewritten by the engineering model.
- **Write commands are explicit about what they touch**: `clear` needs `--yes`; `mark --suggest`
  previews and `--accept-suggested` writes; `digest --dry-run` exists for a reason. Keep that shape.
- **Config keys** are documented in `config.example.yaml` and `ASSAIO_*` env vars in
  `internal/config`; `TestDocumentsPrintNoFlagTheBinaryLacks` reads `README.md`,
  `CONTRIBUTING.md`, `AGENTS.md`, `RELEASING.md` for flags that do not exist.
- Nothing here makes a network call except `sync` and `serve`, both opt-in; keep the offline
  promise (`PRIVACY.md`).
