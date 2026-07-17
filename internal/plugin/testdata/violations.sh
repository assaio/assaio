#!/bin/sh
# Fixture plugin with a valid handshake but several invalid record lines, to exercise
# the skip-and-count boundary validation.
echo '{"assaio_plugin":1,"tool":"demo"}'
echo '{"session_id":"s1","timestamp":"2026-07-01T10:00:00Z","model":"m","input_tokens":100,"output_tokens":200,"dedupe_key":"s1:0","granularity":"turn"}'
echo '{"session_id":"s1","timestamp":"2026-07-01T10:01:00Z","model":"m","input_tokens":-5,"output_tokens":200,"dedupe_key":"s1:1","granularity":"turn"}'
echo '{"session_id":"s1","timestamp":"2026-07-01T10:02:00Z","model":"m","input_tokens":10,"output_tokens":20,"dedupe_key":"","granularity":"turn"}'
echo '{"session_id":"s1","timestamp":"not-a-time","model":"m","input_tokens":10,"output_tokens":20,"dedupe_key":"s1:3","granularity":"turn"}'
echo '{"session_id":"s1","timestamp":"2026-07-01T10:04:00Z","model":"m","input_tokens":10,"output_tokens":20,"dedupe_key":"s1:4","granularity":"weekly"}'
echo 'not even json'
