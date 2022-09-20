#!/usr/bin/env sh
set -e
exec fluentd --config /etc/fluentd/fluentd.conf --no-supervisor --quiet
