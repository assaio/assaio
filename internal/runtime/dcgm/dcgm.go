// Package dcgm knows which metric families NVIDIA's DCGM exporter publishes and what each one
// means. One adapter for one exporter, for the same reason vLLM has its own: the definitions
// are what go stale, and hiding them in a shared Prometheus reader is how a renamed field
// becomes a silently wrong number.
//
// Names are the exporter's own, verbatim:
// https://docs.nvidia.com/datacenter/dcgm/latest/reference/dcgm-exporter-metrics.html
package dcgm

import "github.com/assaio/assaio/internal/runtime"

// Source is the adapter's name in output.
const Source = "dcgm"

// Catalog is the set assaio reads: utilization, memory, power, energy, temperature and health.
func Catalog() []runtime.Capability {
	return []runtime.Capability{
		{
			Key: "gpu-utilization", Metric: "DCGM_FI_DEV_GPU_UTIL",
			Unit: "percent", UnitSource: "DCGM exporter documentation",
			Summary: "Share of time the GPU was busy. Busy is not productive: a GPU spinning on a small batch reads the same as one saturated with work.",
		},
		{
			Key: "memory-used", Metric: "DCGM_FI_DEV_FB_USED",
			Unit: "MiB", UnitSource: "DCGM exporter documentation",
			Summary: "Framebuffer memory in use.",
		},
		{
			Key: "memory-free", Metric: "DCGM_FI_DEV_FB_FREE",
			Unit: "MiB", UnitSource: "DCGM exporter documentation",
			Summary: "Framebuffer memory free.",
		},
		{
			Key: "power-draw", Metric: "DCGM_FI_DEV_POWER_USAGE",
			Unit: "watts", UnitSource: "DCGM exporter documentation",
			Summary: "Instantaneous power draw.",
		},
		{
			Key: "energy-consumed", Metric: "DCGM_FI_DEV_TOTAL_ENERGY_CONSUMPTION",
			Unit: "millijoules", UnitSource: "DCGM exporter documentation",
			Summary: "Energy since the driver last reset. Cumulative: an energy cost needs two reads and the interval between them, which this command does not take.",
		},
		{
			Key: "temperature", Metric: "DCGM_FI_DEV_GPU_TEMP",
			Unit: "celsius", UnitSource: "DCGM exporter documentation",
			Summary: "GPU core temperature.",
		},
		{
			Key: "sm-clock", Metric: "DCGM_FI_DEV_SM_CLOCK",
			Unit: "MHz", UnitSource: "DCGM exporter documentation",
			Summary: "Streaming-multiprocessor clock, which throttling moves.",
		},
		{
			Key: "xid-errors", Metric: "DCGM_FI_DEV_XID_ERRORS",
			Unit: "error code", UnitSource: "DCGM exporter documentation",
			Summary: "The last XID error the driver reported. A code, not a count -- reading it as a number of errors is wrong.",
		},
		{
			Key: "ecc-single-bit", Metric: "DCGM_FI_DEV_ECC_SBE_VOL_TOTAL",
			Unit: "errors", UnitSource: "DCGM exporter documentation",
			Summary: "Correctable ECC errors since boot.",
		},
		{
			Key: "ecc-double-bit", Metric: "DCGM_FI_DEV_ECC_DBE_VOL_TOTAL",
			Unit: "errors", UnitSource: "DCGM exporter documentation",
			Summary: "Uncorrectable ECC errors since boot. Non-zero is hardware trouble, whatever the utilization says.",
		},
	}
}
