#!/bin/sh

set -e

echo "hosts: files dns" > /etc/nsswitch.conf

ipsec start

sleep 2
swanctl --load-all

exec tail -f /var/log/charon.log
