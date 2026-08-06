#!/bin/bash
set -e

if [ ! -f /etc/ipsec.d/certs/server-cert.pem ]; then
    echo "Generating CA and server certificates..."

    openssl genrsa -out /etc/ipsec.d/private/ca-key.pem 4096
    chmod 600 /etc/ipsec.d/private/ca-key.pem

    openssl req -x509 -new -nodes -key /etc/ipsec.d/private/ca-key.pem \
        -sha256 -days 3650 -out /etc/ipsec.d/cacerts/ca-cert.pem \
        -subj "/CN=VPN CA"

    openssl genrsa -out /etc/ipsec.d/private/server-key.pem 2048
    chmod 600 /etc/ipsec.d/private/server-key.pem

    openssl req -new -key /etc/ipsec.d/private/server-key.pem \
        -out /tmp/server.csr -subj "/CN=quomaleque.ru"

    openssl x509 -req -in /tmp/server.csr \
        -CA /etc/ipsec.d/cacerts/ca-cert.pem \
        -CAkey /etc/ipsec.d/private/ca-key.pem \
        -CAcreateserial -out /etc/ipsec.d/certs/server-cert.pem \
        -days 1825 -sha256 \
        -extfile <(printf "keyUsage=digitalSignature,keyEncipherment\nextendedKeyUsage=serverAuth\nsubjectAltName=DNS:quomaleque.ru,DNS:$(hostname -f)")

    rm -f /tmp/server.csr
    echo "Certificates generated."
fi

IFS=',' read -ra DNS_ARR <<< "${STRONGSWAN_DNS:-8.8.8.8,1.1.1.1}"
DNS_FIRST="${DNS_ARR[0]:-8.8.8.8}"
DNS_SECOND="${DNS_ARR[1]:-1.1.1.1}"

IP_START="${STRONGSWAN_IPSEC_POOL_START:-10.10.0.1}"
IP_END="${STRONGSWAN_IPSEC_POOL_END:-10.10.0.254}"
IP_POOL="${IP_START}-${IP_END}"

sed -i "s|@@IPSEC_POOL@@|${IP_POOL}|g" /etc/ipsec.conf
sed -i "s|@@DNS@@|${DNS_FIRST},${DNS_SECOND}|g" /etc/ipsec.conf
sed -i "s|@@RADIUS_SECRET@@|${RADIUS_SECRET}|g" /etc/strongswan.conf

sysctl -w net.ipv4.ip_forward=1
sysctl -w net.ipv6.conf.all.forwarding=1

DEFAULT_IF=$(ip route show default 2>/dev/null | awk '/default/ {print $5}' | head -1)

if [ -n "$DEFAULT_IF" ]; then
    iptables -t nat -A POSTROUTING -o "$DEFAULT_IF" -j MASQUERADE
    iptables -A FORWARD -m state --state RELATED,ESTABLISHED -j ACCEPT
    iptables -A FORWARD -i "$DEFAULT_IF" -j ACCEPT
fi

echo "Starting strongSwan..."
ipsec start --nofork &

echo "Waiting for VICI socket..."
for i in $(seq 1 30); do
    if [ -S /var/run/charon.vici ]; then
        echo "VICI socket ready"
        break
    fi
    sleep 1
done

echo "Starting VICI TCP bridge on port 4502..."
socat TCP-LISTEN:4502,reuseaddr,fork UNIX-CONNECT:/var/run/charon.vici &

wait
