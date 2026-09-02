package plugin

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// describeProtocol is the first half of the protocol-4 metric contract: `<command> describe`
// with nothing on stdin, answered with the same handshake line and one Declaration document.
// It is a second invocation rather than a conversation on one pipe because a plugin that has to
// read a request, answer, and then read again is a plugin whose author has to think about
// flushing -- and the shell and Python skeletons this ships with would each get it wrong in a
// different way. Two one-shot runs cost one extra process per metric per analyze; the envelope
// they size is measured in megabytes.
var describeProtocol = docProtocol{
	kind:      "metric",
	verb:      "describe",
	env:       "ASSAIO_METRIC_PROTOCOL",
	maxStdout: maxMetricStdout,
	handshake: parseMetricHandshake,
}

// describeMetric asks cfg's plugin what it reads and returns the declaration it may be held to.
// Violations are returned alongside the error for the `metrics verify` report, the same shape
// parseMetricResult uses.
func describeMetric(ctx context.Context, cfg Config) (Declaration, []string, error) {
	doc, err := describeProtocol.run(ctx, cfg, nil)
	if err != nil {
		return Declaration{}, nil, err
	}
	declared, violations, err := parseDeclaration(doc)
	if err != nil {
		return Declaration{}, violations,
			fmt.Errorf("metric plugin %s: describe: %w%s", cfg.Name, err, violationSuffix(violations))
	}
	return declared, nil, nil
}

// parseDeclaration decodes one Declaration document and enforces the boundary contract.
// Unknown fields are refused for the same reason the result decoder refuses them: a misspelled
// key that decodes to nothing is a declaration the author believes they made.
func parseDeclaration(doc []byte) (Declaration, []string, error) {
	var d Declaration
	dec := json.NewDecoder(bytes.NewReader(doc))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&d); err != nil {
		return Declaration{}, nil, fmt.Errorf("decoding declaration: %w", err)
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return Declaration{}, nil, errors.New("trailing data after the declaration document")
	}
	if violations := validateDeclaration(d); len(violations) > 0 {
		return Declaration{}, violations, fmt.Errorf("declaration failed %d contract check(s)", len(violations))
	}
	return d, nil, nil
}
