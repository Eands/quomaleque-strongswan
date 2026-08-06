FROM ubuntu:22.04

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && apt-get install -y \
    freeradius \
    freeradius-postgresql \
    freeradius-utils \
    && rm -rf /var/lib/apt/lists/*

COPY clients.conf /etc/freeradius/3.0/clients.conf
COPY mods-enabled/eap /etc/freeradius/3.0/mods-enabled/eap
COPY mods-enabled/sql /etc/freeradius/3.0/mods-enabled/sql
COPY sites-enabled/default /etc/freeradius/3.0/sites-enabled/default
COPY mods-config/sql/main/postgresql/queries.conf /etc/freeradius/3.0/mods-config/sql/main/postgresql/queries.conf
COPY mods-config/sql/counter/postgresql/max_connections.conf /etc/freeradius/3.0/mods-config/sql/counter/postgresql/max_connections.conf
COPY scripts/entrypoint.sh /entrypoint.sh

RUN chmod +x /entrypoint.sh && \
    mkdir -p /var/log/freeradius && \
    chown freerad:adm /var/log/freeradius && \
    chmod 750 /var/log/freeradius && \
    ln -sf /etc/freeradius/3.0/mods-config/sql/counter/postgresql/max_connections.conf \
           /etc/freeradius/3.0/mods-enabled/max_connections

EXPOSE 1812/udp 1813/udp

ENTRYPOINT ["/entrypoint.sh"]
