#!/bin/sh
# Fixture plugin that hangs past any reasonable test timeout, to exercise the context
# deadline kill path.
sleep 30
echo '{"assaio_plugin":1,"tool":"demo"}'
