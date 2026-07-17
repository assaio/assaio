# 1. Language and core architecture

## Status
Accepted (2026-07-11)

## Context
assaio needs a component that runs on every developer machine (log parsing, hooks) with
zero runtime dependencies, and a server component, both cross-compiled to mac/linux/windows.

## Decision
Go. Two binaries from one module: `assaio` (server, later stages) and `assaio-agent`.
Embedded SQLite (modernc, no CGO) by default; Postgres optional at scale. Connectors,
metrics, and rules are self-contained units behind stable interfaces. Apache-2.0 core;
commercial modules isolated under `ee/`.

## Consequences
Single static binaries, trivial cross-compilation, no CGO. The agent MVP keeps log
readers in `internal/parser/<vendor>/`; the plugin registry arrives with the server.
