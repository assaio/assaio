#!/bin/sh
# Happy-path rule plugin: checks argv/env/stdin contract, emits handshake + alerts.
[ "$1" = "evaluate" ] || exit 64
[ "$ASSAIO_RULE_PROTOCOL" = "1" ] || exit 64
grep -q '"assaio_rule_input":1' || exit 65
echo '{"assaio_rule":1,"name":"demo"}'
echo '{"alerts":[{"rule":"token-spike","severity":"warn","message":"Tokens up sharply.","validator":"burn-anomaly"},{"rule":"cache-cold","severity":"info","message":"Cache reuse is low."}]}'
