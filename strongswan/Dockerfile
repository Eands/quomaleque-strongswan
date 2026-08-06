FROM ubuntu:22.04

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && apt-get install -y \
    strongswan \
    strongswan-swanctl \
    libcharon-extra-plugins \
    libstrongswan-extra-plugins \
    socat \
    iptables \
    ca-certificates \
    tcpdump \
    && rm -rf /var/lib/apt/lists/*

RUN mkdir -p /etc/ipsec.d/certs /etc/ipsec.d/private /etc/ipsec.d/cacerts

COPY ipsec.conf /etc/ipsec.conf
COPY strongswan.conf /etc/strongswan.conf
COPY ipsec.secrets /etc/ipsec.secrets
COPY scripts/entrypoint.sh /entrypoint.sh

RUN chmod +x /entrypoint.sh

EXPOSE 500/udp 4500/udp

ENTRYPOINT ["/entrypoint.sh"]
