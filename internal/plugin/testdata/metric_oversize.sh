#!/bin/sh
cat >/dev/null
echo '{"assaio_metric":2,"name":"demo"}'
head -c 2097152 /dev/zero | tr '\0' 'a'
