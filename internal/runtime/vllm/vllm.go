// Package vllm knows which metric families a vLLM OpenAI-compatible server exposes and what
// each one means. It is one adapter for one runtime: a universal Prometheus reader would put
// every runtime's metric definitions in one file, and the definitions are the part that goes
// stale.
//
// Names are vLLM's own, verbatim, so they can be grepped against its documentation:
// https://docs.vllm.ai/en/latest/design/metrics/
package vllm

import "github.com/assaio/assaio/internal/runtime"

// Source is the adapter's name in output.
const Source = "vllm"

// Catalog is what assaio looks for. It is deliberately a short list of families that answer a
// question a reader has -- how loaded is this deployment, is it queueing, is the cache full,
// how fast is it answering -- rather than every metric vLLM publishes. A missing entry here is
// a capability assaio does not read, which the snapshot reports as such.
func Catalog() []runtime.Capability {
	return []runtime.Capability{
		{
			Key: "running-requests", Metric: "vllm:num_requests_running",
			Unit: "requests", UnitSource: "vLLM documentation",
			Summary: "Requests the engine is decoding right now.",
		},
		{
			Key: "waiting-requests", Metric: "vllm:num_requests_waiting",
			Unit: "requests", UnitSource: "vLLM documentation",
			Summary: "Requests admitted and queued, waiting for a slot. Non-zero with idle GPUs is the shape of a scheduling limit rather than a hardware one.",
		},
		{
			Key: "kv-cache-usage", Metric: "vllm:gpu_cache_usage_perc",
			Unit: "fraction 0-1", UnitSource: "vLLM documentation",
			Summary: "Share of the GPU KV cache in use. Near 1 with requests waiting is the classic capacity ceiling.",
		},
		{
			Key: "prefix-cache-hits", Metric: "vllm:gpu_prefix_cache_hits",
			Unit: "blocks", UnitSource: "vLLM documentation",
			Summary: "Prefix-cache block hits since start, beside the queries below.",
		},
		{
			Key: "prefix-cache-queries", Metric: "vllm:gpu_prefix_cache_queries",
			Unit: "blocks", UnitSource: "vLLM documentation",
			Summary: "Prefix-cache block lookups since start. The hit rate is hits over queries -- both cumulative, so it is a lifetime rate, not a current one.",
		},
		{
			Key: "prompt-tokens", Metric: "vllm:prompt_tokens",
			Unit: "tokens", UnitSource: "vLLM documentation",
			Summary: "Prompt tokens processed since start.",
		},
		{
			Key: "generation-tokens", Metric: "vllm:generation_tokens",
			Unit: "tokens", UnitSource: "vLLM documentation",
			Summary: "Tokens generated since start.",
		},
		{
			Key: "requests-finished", Metric: "vllm:request_success",
			Unit: "requests", UnitSource: "vLLM documentation",
			Summary: "Requests that finished, labelled by the reason they stopped -- a stop token, the length ceiling, or an abort.",
		},
		{
			Key: "time-to-first-token", Metric: "vllm:time_to_first_token_seconds",
			Unit: "seconds", UnitSource: "vLLM documentation",
			Summary: "Time-to-first-token distribution. Only the observation count is read here; a percentile off one snapshot describes the whole life of the process.",
		},
		{
			Key: "time-per-output-token", Metric: "vllm:time_per_output_token_seconds",
			Unit: "seconds", UnitSource: "vLLM documentation",
			Summary: "Inter-token latency distribution, read the same way and with the same limit.",
		},
		{
			Key: "e2e-request-latency", Metric: "vllm:e2e_request_latency_seconds",
			Unit: "seconds", UnitSource: "vLLM documentation",
			Summary: "End-to-end request latency distribution.",
		},
		{
			Key: "preemptions", Metric: "vllm:num_preemptions",
			Unit: "requests", UnitSource: "vLLM documentation",
			Summary: "Requests preempted and rescheduled since start -- the engine giving a slot back under pressure.",
		},
	}
}
