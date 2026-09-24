# Runtime inspect (experimental)

`assaio-agent runtime inspect` reads one snapshot of metrics published by a self-hosted inference
deployment. It is read-only and stores nothing.

**This is a feasibility slice, not a monitoring product.** It tests whether runtime evidence
alongside assaio's usage evidence changes a real decision. Until three self-hosted deployments
confirm that, the feature is not promised and may be removed. See [ROADMAP.md](../ROADMAP.md) for
the gate and kill criterion.

## What it can and cannot see

It sees only infrastructure **you** operate. Hosted Claude, Codex, Gemini, and Cursor sessions run
on vendor accelerators that local signals cannot reveal; assaio reports them as `unknown` rather
than estimating. Coding-tool activity and runtime telemetry remain separate. Joining them requires a
request, trace, deployment, or workload identity that neither currently carries.

It does not store time series, estimate self-hosted cost, recommend GPU or runtime changes, or
change configuration. It reads no prompts, responses, code, or diffs because its endpoints contain
none.

## Running it

Against live endpoints:

```sh
assaio-agent runtime inspect \
  --vllm-url http://127.0.0.1:8000/metrics \
  --dcgm-url http://127.0.0.1:9400/metrics
```

Against saved snapshots; this is deterministic and needs no deployment:

```sh
curl -s http://gpu-node:8000/metrics > vllm.prom
curl -s http://gpu-node:9400/metrics > dcgm.prom
assaio-agent runtime inspect --vllm-file vllm.prom --dcgm-file dcgm.prom
```

`--vllm-url` and `--vllm-file` cannot be used together; the same applies to the DCGM pair. A live
endpoint and a saved snapshot make different claims about a number's source. Either source can be
used alone. `--format json` prints the same content as a deterministic document: identical input
encodes identically. Its format is not a frozen contract while the demand gate remains open.

The flags set explicit bounds: `--timeout` (5s), `--max-bytes` (8 MiB), and `--max-redirects` (2;
`0` blocks all redirects). Requests are plain GETs without headers, bodies, or credentials. An
inspection that carries secrets would need a threat model this experiment lacks.

## What the output means

The output lists every catalog capability, including those the deployment did not publish. Listing
only found metrics would imply complete coverage.

- **Unavailable is not zero.** An unpublished metric was not measured at zero; check for an exporter
  flag before concluding GPUs are idle.
- **A counter is never a rate.** Counters accumulate from exporter startup. Throughput requires a
  second reading and the interval; this command takes one reading and labels every counter
  accordingly.
- **A histogram reports its observation count only.** A percentile from one snapshot's buckets
  covers the process's whole lifetime, not recent activity.
- **Units come from the exporter where it declares one** (`# UNIT`); otherwise they come from
  assaio's catalog, based on vendor documentation. The output names the source.
- **Unreachable establishes nothing.** A failed read is reported as a failure, not as a deployment
  with no metrics.
- **Partial stays partial.** Skipped lines and truncated reads appear before missing capabilities
  because either makes those absences unproven.

## What it reads

Metric names match the vendors' names exactly, so you can find them in the
[vLLM](https://docs.vllm.ai/en/latest/design/metrics/) and [NVIDIA DCGM
exporter](https://docs.nvidia.com/datacenter/dcgm/latest/reference/dcgm-exporter-metrics.html)
documentation. `internal/runtime/vllm` and `internal/runtime/dcgm` each hold one exporter catalog.
Separate adapters keep definitions visible when vendor names change; a shared Prometheus reader
could silently report a wrong number.

**Test fixtures come from those documents, not a running deployment.** They prove the parser handles
the documented format, not what any specific version emits. A redacted snapshot from a real
deployment would address this limit; see `internal/runtime/testdata/README.md`.
