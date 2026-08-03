#!/bin/sh

set -e

ipsec start

while [ ! -S /var/run/charon.vici ]; do
    sleep 1
done

swanctl --load-all

exec tail -f /dev/null
