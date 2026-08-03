FROM ubuntu:24.04

RUN apt-get update && apt-get install -y \
    strongswan \
    strongswan-swanctl \
    libcharon-extra-plugins \
    iptables \
    && rm -rf /var/lib/apt/lists/*

RUN mkdir -p /etc/swanctl/x509 \
    /etc/swanctl/x509ca \
    /etc/swanctl/private

COPY config/strongswan.conf /etc/strongswan.conf
COPY config/swanctl/ /etc/swanctl/

EXPOSE 500/udp 4500/udp

CMD sh -c 'swanctl --load-all && ipsec start --nofork'
