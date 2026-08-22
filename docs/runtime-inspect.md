# Runtime inspect (experimental)

`assaio-agent runtime inspect` reads the metrics a self-hosted inference deployment already
publishes — one snapshot, read-only, nothing stored.

**This is a feasibility slice, not a monitoring product.** It exists to find out whether runtime
evidence beside assaio's usage evidence changes a decision anybody actually makes. Until three
self-hosted deployments say it does, nothing here is a promise and it may be removed. See
[ROADMAP.md](../ROADMAP.md) for the gate and its kill criterion.

## What it can and cannot see

It can only see infrastructure **you** operate. A hosted Claude, Codex, Gemini or Cursor session
runs on the vendor's accelerators, and no local signal reveals them: assaio leaves that
`unknown` rather than estimating it. Coding-tool activity and runtime telemetry are separate
evidence planes and this command does not join them — joining needs a request, trace, deployment
or workload identity that neither side currently carries.

It does not: store a time series, estimate a self-hosted cost, recommend a GPU or runtime
change, or alter any configuration. It reads no prompt, response, code or diff, because the
endpoints it reads carry none.

## Running it

Against live endpoints:

```sh
assaio-agent runtime inspect \
  --vllm-url http://127.0.0.1:8000/metrics \
  --dcgm-url http://127.0.0.1:9400/metrics
```

Against saved snapshots, which is deterministic and needs no deployment:

```sh
curl -s http://gpu-node:8000/metrics > vllm.prom
curl -s http://gpu-node:9400/metrics > dcgm.prom
assaio-agent runtime inspect --vllm-file vllm.prom --dcgm-file dcgm.prom
```

`--vllm-url` and `--vllm-file` are mutually exclusive, as are the DCGM pair: a live endpoint and
a saved snapshot are two different claims about where a number came from. Either source may be
used alone. `--format json` prints the same content as a deterministic document -- the same input encodes the same way -- and not as a frozen contract while the demand gate is open.

Bounds are flags, not hidden defaults: `--timeout` (5s), `--max-bytes` (8 MiB), `--max-redirects`
(2; `0` forbids redirects entirely). The request is a plain GET with no header, body or
credential — an inspection that could carry a secret would need a threat model this experiment
does not have.

## What the output means

Every capability in the catalog appears whether or not the deployment published it. An output
listing only what it found would read as complete.

- **unavailable is not zero.** A metric nobody published is not a metric measured at zero, and
  the difference decides whether you go looking for an exporter flag or conclude your GPUs are
  idle.
- **A counter is never a rate.** Counters here are cumulative since the exporter started.
  Turning one into a throughput needs a second read and the interval between them; this command
  takes one read and says so on every counter it prints.
- **A histogram reports its observation count only.** A percentile computed from one snapshot's
  buckets describes the whole life of the process, not "lately".
- **Units come from the exporter where it declares one** (`# UNIT`), and otherwise from assaio's
  catalog, which was read off the vendor's documentation. The output says which.
- **Unreachable establishes nothing.** A failed read is reported as a failed read, never as a
  deployment that exposes nothing.
- **Partial stays partial.** Skipped lines and a truncated read are reported *before* the list of
  missing capabilities, because either one makes every absence in that list unproven.

## What it reads

Metric names are the vendors' own, verbatim, so they can be grepped against the documentation
they come from: [vLLM](https://docs.vllm.ai/en/latest/design/metrics/) and the
[NVIDIA DCGM exporter](https://docs.nvidia.com/datacenter/dcgm/latest/reference/dcgm-exporter-metrics.html).
`internal/runtime/vllm` and `internal/runtime/dcgm` hold one catalog each — one adapter per
exporter, because the definitions are the part that goes stale, and hiding them in a shared
Prometheus reader is how a renamed field becomes a silently wrong number.

**The test fixtures are constructed from those documents, not captured from a running
deployment.** That is a real limitation: a constructed fixture proves the parser reads the shape
the documentation describes and proves nothing about the shape a particular version emits. A
redacted snapshot from a real deployment is the contribution this needs — see
`internal/runtime/testdata/README.md`.
