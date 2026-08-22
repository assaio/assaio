#!/bin/sh
# Happy-path metric plugin: checks argv/env/stdin contract, emits handshake + Result.
[ "$1" = "analyze" ] || exit 64
[ "$ASSAIO_METRIC_PROTOCOL" = "1" ] || exit 64
grep -q '"assaio_metric_input":3' || exit 65
echo '{"assaio_metric":3,"name":"demo"}'
echo '{"title":"Demo Metric","layer":"activity","read":{"key":"watch","label":"WATCH"},"purity":0.4,"howToRead":"Directional demo.","figures":[{"label":"x","value":"1"}],"takeaway":"Demo takeaway.","confidence":{"samples":12,"samplesUnit":"sessions"}}'
