#!/bin/sh
if [ "$1" = "describe" ]; then
  echo '{"assaio_metric":4,"name":"demo"}'
  echo '{"needs":["usage"]}'
  exit 0
fi
cat >/dev/null
echo '{"assaio_metric":4,"name":"demo"}'
echo '{"title":"T","layer":"activity","read":{"key":"great","label":"GREAT"},"howToRead":"H","takeaway":"K"}'
