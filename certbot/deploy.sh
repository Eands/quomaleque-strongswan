#!/bin/sh

set -e

DOMAIN="quomaleque.ru"
SRC="/etc/letsencrypt/live/$DOMAIN"
DST="/certs"

if [ ! -d "$SRC" ]; then
    echo "Certificate directory $SRC not found — run certonly first"
    exit 0
fi

echo "Deploying certs for $DOMAIN..."

cp "$SRC/fullchain.pem" "$DST/serverCert.pem"
cp "$SRC/privkey.pem"   "$DST/serverKey.pem"

echo "Certs deployed"
echo "Run: docker compose --profile prod restart strongswan-prod nginx"
