#!/bin/sh

set -e

DOMAIN="quomaleque.ru"
SRC="/etc/letsencrypt/live/$DOMAIN"
DST="/swanctl"

if [ ! -d "$SRC" ]; then
    echo "Certificate directory $SRC not found — run certonly first"
    exit 0
fi

echo "Deploying certs for $DOMAIN..."

cp "$SRC/fullchain.pem" "$DST/x509/serverCert.pem"
cp "$SRC/privkey.pem"   "$DST/private/serverKey.pem"
cp "$SRC/chain.pem"     "$DST/x509ca/caCert.pem"

echo "Certs deployed to swanctl directory"
echo "Run: docker compose restart strongswan nginx"
