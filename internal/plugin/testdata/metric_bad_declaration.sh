#!/bin/sh
# Declares a capability this build does not know: the describe run must be refused whole rather
# than reduced to the names it happens to recognize.
if [ "$1" = "describe" ]; then
  echo '{"assaio_metric":4,"name":"demo"}'
  echo '{"needs":["usage","telemetry"]}'
  exit 0
fi
cat >/dev/null
echo '{"assaio_metric":4,"name":"demo"}'
echo '{"title":"T","layer":"activity","read":{"key":"good","label":"OK"},"howToRead":"H","takeaway":"K"}'
