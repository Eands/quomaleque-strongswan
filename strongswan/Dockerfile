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
COPY config/swanctl/swanctl.conf /etc/swanctl/swanctl.conf
COPY config/swanctl/x509/ /etc/swanctl/x509/
COPY config/swanctl/x509ca/ /etc/swanctl/x509ca/
COPY config/swanctl/private/ /etc/swanctl/private/
COPY startup.sh /

EXPOSE 500/udp 4500/udp

ENTRYPOINT ["/startup.sh"]
