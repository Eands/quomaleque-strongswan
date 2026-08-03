#!/bin/sh

set -e

C=US
O=quomaleque.ru
CA_CN=quomaleque.ru
SERVER_CN=quomaleque.ru
SERVER_SAN=quomaleque.ru
CLIENT_CN="quomaleque.ru"

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
CONFIG_DIR="$SCRIPT_DIR/../config/swanctl"

mkdir -p "$CONFIG_DIR/x509ca" \
         "$CONFIG_DIR/x509" \
         "$CONFIG_DIR/private"

pki --gen --outform pem > "$CONFIG_DIR/private/caKey.pem"
pki --self --in "$CONFIG_DIR/private/caKey.pem" --dn "C=$C, O=$O, CN=$CA_CN" --ca --outform pem > "$CONFIG_DIR/x509ca/caCert.pem"

pki --gen --outform pem > "$CONFIG_DIR/private/serverKey.pem"
pki --issue --in "$CONFIG_DIR/private/serverKey.pem" --type priv --cacert "$CONFIG_DIR/x509ca/caCert.pem" --cakey "$CONFIG_DIR/private/caKey.pem" --dn "C=$C, O=$O, CN=$SERVER_CN" --san="$SERVER_SAN" --flag serverAuth --flag ikeIntermediate --outform pem > "$CONFIG_DIR/x509/serverCert.pem"

pki --gen --outform pem > "$CONFIG_DIR/private/clientKey.pem"
pki --issue --in "$CONFIG_DIR/private/clientKey.pem" --type priv --cacert "$CONFIG_DIR/x509ca/caCert.pem" --cakey "$CONFIG_DIR/private/caKey.pem" --dn "C=$C, O=$O, CN=$CLIENT_CN" --san="$CLIENT_CN" --outform pem > "$CONFIG_DIR/x509/clientCert.pem"
openssl pkcs12 -export -inkey "$CONFIG_DIR/private/clientKey.pem" -in "$CONFIG_DIR/x509/clientCert.pem" -name "$CLIENT_CN" -certfile "$CONFIG_DIR/x509ca/caCert.pem" -caname "$CA_CN" -out "$CONFIG_DIR/clientCert.p12"

echo "Certificates generated successfully in $CONFIG_DIR"
