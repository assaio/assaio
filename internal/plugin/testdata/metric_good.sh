#!/bin/sh
# Happy-path metric plugin: checks argv/env/stdin contract, declares what it reads, emits
# handshake + Result.
[ "$ASSAIO_METRIC_PROTOCOL" = "1" ] || exit 64
if [ "$1" = "describe" ]; then
  echo '{"assaio_metric":4,"name":"demo"}'
  echo '{"needs":["usage"],"fields":{"usage":["day","tool","in","out"]}}'
  exit 0
fi
[ "$1" = "analyze" ] || exit 64
grep -q '"assaio_metric_input":4' || exit 65
echo '{"assaio_metric":4,"name":"demo"}'
echo '{"title":"Demo Metric","layer":"activity","read":{"key":"watch","label":"WATCH"},"purity":0.4,"howToRead":"Directional demo.","figures":[{"label":"x","value":"1"}],"takeaway":"Demo takeaway.","confidence":{"samples":12,"samplesUnit":"sessions"}}'
