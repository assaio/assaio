# Runtime fixtures

**These are constructed, not captured.** Every other golden file in this repository is a real
sample from a real tool; these two are written from the vendors' published metric documentation
because nobody has yet contributed a snapshot from a running deployment.

That is a real limitation and it is stated here rather than in a commit message: a constructed
fixture proves the parser reads the shape the documentation describes, and proves nothing about
the shape a particular version actually emits. Field order, label sets, extra families and
renamed metrics are exactly what a capture would settle.

A redacted `curl http://<host>:8000/metrics` from a real vLLM server, or `curl
http://<host>:9400/metrics` from a real DCGM exporter, is the contribution this needs. Metric
values are not sensitive; hostnames, pod names and UUIDs in the labels are, and can be replaced
with anything before sending.

Sources the names come from:

- vLLM — <https://docs.vllm.ai/en/latest/design/metrics/>
- DCGM exporter — <https://docs.nvidia.com/datacenter/dcgm/latest/reference/dcgm-exporter-metrics.html>
