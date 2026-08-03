#!/bin/sh

set -e

rm -f /var/run/charon.vici

ipsec start

while [ ! -S /var/run/charon.vici ]; do
    sleep 1
done

swanctl --load-all

exec tail -f /var/log/charon.log
