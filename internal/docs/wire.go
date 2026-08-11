package docs

import "github.com/assaio/assaio/internal/plugin"

// MetricWire is the metric-plugin protocol (ADR 0004) as an enumeration, re-exported from the
// package that owns the shape. What each field *means* stays prose in docs/extending.md:
// reflection reads names and types, never doc comments.
type MetricWire = plugin.MetricWire

// WireField is one field of that protocol.
type WireField = plugin.WireField

func metricWire() MetricWire { return plugin.MetricWireFields() }
