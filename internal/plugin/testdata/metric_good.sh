#!/bin/sh
# Happy-path metric plugin: checks argv/env/stdin contract, emits handshake + Result.
[ "$1" = "analyze" ] || exit 64
[ "$ASSAIO_METRIC_PROTOCOL" = "1" ] || exit 64
grep -q '"assaio_metric_input":1' || exit 65
echo '{"assaio_metric":1,"name":"demo"}'
echo '{"title":"Demo Metric","read":{"key":"watch","label":"WATCH"},"purity":0.4,"howToRead":"Directional demo.","figures":[{"label":"x","value":"1"}],"takeaway":"Demo takeaway."}'
