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
PKI="docker run --rm --entrypoint pki -v $CONFIG_DIR:/etc/swanctl jesusdf/docker-strongswan"

mkdir -p "$CONFIG_DIR/x509ca" \
         "$CONFIG_DIR/x509" \
         "$CONFIG_DIR/private"

$PKI --gen --outform pem > "$CONFIG_DIR/private/caKey.pem"
$PKI --self --in /etc/swanctl/private/caKey.pem --dn "C=$C, O=$O, CN=$CA_CN" --ca --outform pem > "$CONFIG_DIR/x509ca/caCert.pem"

$PKI --gen --outform pem > "$CONFIG_DIR/private/serverKey.pem"
$PKI --issue --in /etc/swanctl/private/serverKey.pem --type priv --cacert /etc/swanctl/x509ca/caCert.pem --cakey /etc/swanctl/private/caKey.pem --dn "C=$C, O=$O, CN=$SERVER_CN" --san="$SERVER_SAN" --flag serverAuth --flag ikeIntermediate --outform pem > "$CONFIG_DIR/x509/serverCert.pem"

$PKI --gen --outform pem > "$CONFIG_DIR/private/clientKey.pem"
$PKI --issue --in /etc/swanctl/private/clientKey.pem --type priv --cacert /etc/swanctl/x509ca/caCert.pem --cakey /etc/swanctl/private/caKey.pem --dn "C=$C, O=$O, CN=$CLIENT_CN" --san="$CLIENT_CN" --outform pem > "$CONFIG_DIR/x509/clientCert.pem"
openssl pkcs12 -export -inkey "$CONFIG_DIR/private/clientKey.pem" -in "$CONFIG_DIR/x509/clientCert.pem" -name "$CLIENT_CN" -certfile "$CONFIG_DIR/x509ca/caCert.pem" -caname "$CA_CN" -out "$CONFIG_DIR/clientCert.p12"

echo "Certificates generated successfully in $CONFIG_DIR"
