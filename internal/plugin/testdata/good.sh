#!/bin/sh
# Well-behaved fixture plugin: valid handshake, two valid records.
echo '{"assaio_plugin":1,"tool":"demo"}'
echo '{"session_id":"s1","timestamp":"2026-07-01T10:00:00Z","model":"some-model","input_tokens":100,"output_tokens":200,"cache_read_tokens":0,"cache_write_tokens":0,"reasoning_tokens":0,"dedupe_key":"s1:0","project":"myrepo","git_branch":"main","entrypoint":"cli","granularity":"turn"}'
echo '{"session_id":"s1","timestamp":"2026-07-01T10:05:00Z","model":"some-model","input_tokens":50,"output_tokens":75,"cache_read_tokens":10,"cache_write_tokens":0,"reasoning_tokens":0,"dedupe_key":"s1:1","granularity":"turn"}'
echo 'diagnostic line on stderr' 1>&2
