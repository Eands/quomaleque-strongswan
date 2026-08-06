FROM freeradius/freeradius-server:latest

COPY clients.conf /etc/raddb/clients.conf
COPY mods-enabled/eap /etc/raddb/mods-enabled/eap
COPY mods-enabled/sql /etc/raddb/mods-enabled/sql
COPY sites-enabled/default /etc/raddb/sites-enabled/default
COPY mods-config/sql/main/postgresql/queries.conf /etc/raddb/mods-config/sql/main/postgresql/queries.conf
COPY mods-config/sql/counter/postgresql/max_connections.conf /etc/raddb/mods-config/sql/counter/postgresql/max_connections.conf
COPY scripts/entrypoint.sh /entrypoint.sh

RUN chmod +x /entrypoint.sh && ln -sf /etc/raddb/mods-config/sql/counter/postgresql/max_connections.conf /etc/raddb/mods-enabled/max_connections

EXPOSE 1812/udp 1813/udp

ENTRYPOINT ["/entrypoint.sh"]
