// Package analyze is assaio's validator framework. Each Validator reads the same Input
// bundle and returns one structured Result -- the single source of truth the CLI text
// report and HTML dashboard both render from. Adding a metric is a one-file change:
// implement Validator and call Register from that file's init() -- see "Adding a metric
// validator" in docs/extending/metric-validator.md.
package analyze
