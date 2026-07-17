#!/bin/sh
# Second well-behaved fixture plugin (tool name "good"), used where a test needs two
# distinct configured plugins running side by side.
echo '{"assaio_plugin":1,"tool":"good"}'
echo '{"session_id":"s1","timestamp":"2026-07-01T10:00:00Z","model":"some-model","input_tokens":100,"output_tokens":200,"dedupe_key":"s1:0","granularity":"turn"}'
echo '{"session_id":"s1","timestamp":"2026-07-01T10:05:00Z","model":"some-model","input_tokens":50,"output_tokens":75,"dedupe_key":"s1:1","granularity":"turn"}'
