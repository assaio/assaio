---
name: content-model
description: When and how text goes through the GPT/Gemini door (.claude/scripts/text_model.py) instead of being written by the engineering model — README and docs prose, i18n and help strings, changelog wording, site copy, and any audit or verdict about such text; also the second-family challenge of a research note. Use "rewrite this sentence", "review the wording", "simplify the docs", "second opinion".
argument-hint: "<what> [--engine auto|gpt|gemini|both] [--effort low|medium|high]"
allowed-tools: Bash, Read, Write, Glob, Agent
---

Request: $ARGUMENTS

## The rule

Anthropic models engineer here. Text that reaches a reader — and every verdict about such
text — comes from GPT (`codex`) or Gemini (`agy`) through `.claude/scripts/text_model.py`.
No engine → exit 2 → stop and report; never fill the gap yourself. This does not cover code,
comments, commit messages or this harness.

## How

Delegate the run to `text-broker` with: the prompt, the input file(s), the schema when the
answer must be checkable, the engine and effort. One call per batch of 10–30 items, never per
item. Before more than 5 calls, state the count and wait.

| You want | Engine | Cost |
|---|---|---|
| a draft or a rewrite of a few sentences | `auto` | 1 call |
| a verdict that keeps / rewrites / drops items | `both`, with a schema | 2 calls; a removal counts only where both agree |
| the strongest case against a plan or a note | `both`, effort `high` | 2 calls |
| structured extraction (ids in, ids out) | `gpt` or `gemini`, with a schema | 1 call |

The isolation is the door's job: `codex exec --sandbox read-only --ephemeral` in an empty
working directory, `agy --sandbox`; only the files you pass as `--input` are seen. Models: GPT
from the user's codex config (`TEXT_MODEL_GPT` overrides), Gemini `gemini-3.1-pro-<effort>`
(`TEXT_MODEL_GEMINI` overrides, `gemini-*` only).

## What comes back

Mechanically validated (parses, every id returned, placeholders and backticks intact, lengths
hold) — then it enters the product through the normal gates: `make docs`, `make test` (the
i18n catalog and the golden dashboards), `/review` for the surfaces. A rewrite of a sentence a
`data-claim` span covers still has to keep the number the test checks.
