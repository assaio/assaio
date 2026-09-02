# Support

Four kinds of message, four different places. Picking the right one is the whole point of
this page.

| You have | Go to |
|---|---|
| A question, or "is this number right?" | [Discussions](https://github.com/assaio/assaio/discussions) |
| Something is broken | a [bug report](https://github.com/assaio/assaio/issues/new?template=bug.yml) |
| A security vulnerability | [`SECURITY.md`](SECURITY.md) — **never** a public issue |
| An idea for a new capability | [`BACKLOG.md`](BACKLOG.md) first, then a [feature request](https://github.com/assaio/assaio/issues/new?template=feature.yml) |

Blank issues are disabled on purpose: every one of the routes above is faster than an
untriaged issue.

## Before you file: the answers assaio already has

Most "why is this number zero?" questions are answered by the binary itself, and answering
them that way takes a minute rather than a round trip.

- **`assaio-agent doctor`** — which tools were detected, where their logs were found, the
  store's size and freshness, what is unpriced, and whether a format-drift canary fired. Paste
  its output into any bug report; it is the single most useful thing you can attach.
- **`assaio-agent signals`** — what assaio can report, what each figure means, and which of
  your sources can answer it. A figure that reads zero because a source never records it is a
  different fact from one that is genuinely zero, and this is where the difference is written
  down.
- **`assaio-agent demo`** — the full reports on bundled sample data, if you want to see what
  a working run looks like.
- [`docs/reconcile.md`](docs/reconcile.md) — "the cost doesn't match my bill." It is an
  estimate at public pay-as-you-go prices; that page explains how to compare it against a
  vendor's own export and what no export can answer.
- [`docs/format-resilience.md`](docs/format-resilience.md) — a tool shipped an update and
  numbers moved. That page describes the report → fixture → patch-release loop, and issues of
  this kind carry the `format-drift` label.
- [`docs/README.md`](docs/README.md) — the map of everything else.

## Questions

Use [Discussions](https://github.com/assaio/assaio/discussions) — how something works,
whether a reading means what you think it means, what you built on top of it. Questions filed
as issues get moved there, which only slows the answer down.

## Bugs

Use the [bug template](https://github.com/assaio/assaio/issues/new?template=bug.yml). It asks
for the version (`assaio-agent version`), the tool and platform, and what you expected versus
what happened. Add the `doctor` output.

A **wrong number** is a bug of its own kind, and the most valuable report this project
receives — say which figure, what you believe it should be, and how you know. Redact freely:
project names and paths are never needed to reproduce one.

## Security

[`SECURITY.md`](SECURITY.md) has the reporting channel, the acknowledgement and fix
timelines, and what is in scope. Do not open a public issue for a vulnerability.
[`docs/threat-model.md`](docs/threat-model.md) describes what each surface is trusted with,
which is often enough to tell a design boundary from a defect.

## Ideas and new capabilities

[`BACKLOG.md`](BACKLOG.md) is the ranked pool of candidate items with stable ids, and its
header describes the intake rules. Skim it first — your idea may already be `B47` — then open
a [feature request](https://github.com/assaio/assaio/issues/new?template=feature.yml).

Support for a **new AI coding tool** has its own template and its own intake path: open a
[connector issue](https://github.com/assaio/assaio/issues/new?template=connector.yml) before
writing code, and read
[the intake flow](docs/extending/data-source.md#the-intake-path-open-a-connector-issue-first).
A tool only your organization uses is usually better served by an out-of-tree
[exec plugin](docs/extending/parser-plugin.md), which needs no PR at all.

Want to build it yourself? [`CONTRIBUTING.md`](CONTRIBUTING.md) is the authoritative set of
rules, and agreeing the approach in an issue first is expected for anything larger than a fix.

## What to expect

assaio is pre-1.0 and maintained by one person ([`GOVERNANCE.md`](GOVERNANCE.md)). There is
no response-time commitment outside the disclosure timeline in [`SECURITY.md`](SECURITY.md).
A clear bug report with `doctor` output attached is the fastest path to a fix, and a wrong
number goes to the front of the queue.
