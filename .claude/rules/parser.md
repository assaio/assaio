---
paths:
  - "internal/parser/**"
  - "internal/ingest/**"
  - "internal/usage/**"
---

# Parsers and ingest

- **Contract first** (`docs/extending/data-source.md`): single-root `Discover(root)`, skip-and-count
  on a corrupt line, the shared scanner with `MaxLineBytes`, `NonNeg` on every count, a
  deterministic `DedupeKey`. Re-import must be idempotent; a key that drifts duplicates history.
- **Golden files come from real captures**, regenerated with `-update` only when the change meant
  to move the output, and the diff read. Two sources (Gemini, Cline) are calibrated against a
  constructed sample — a claim about them from this machine is not evidence (`B144`).
- **Every parser ships `FuzzParse`** (Cline: `FuzzParseTask`, agy: `FuzzParseTranscript`) with a seed
  corpus; `make fuzz` is part of the gate for any change here.
- **Count once per response, not per content block.** The flagship parser was 2× wrong for eleven
  releases and every guard said fine (`docs/corrections.md`). Ask what the test would look like
  if the reading were wrong, and write that test.
- **A field the source does not record stays empty and declared unanswered** (ADR 0011, 0017):
  never a zero, never averaged in. Activity-only sources (agy) carry no token figure anywhere.
- **Parsers are hermetic**: project resolution is ingest's job (`internal/projectid`); a parser
  never reads git or the network.
- **PRIVACY.md changes in the same commit** as any parser that reads a new directory or field, and
  when a parser reads *less* than the others. `CITATION.cff`'s abstract and the source list in
  `README.md`, `site/llms.txt` and `AGENTS.md` name every source; `consistency.yml` checks the first.
