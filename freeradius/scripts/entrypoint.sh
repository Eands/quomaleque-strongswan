#!/bin/sh
set -e

echo "Waiting for PostgreSQL at ${DB_HOST}:${DB_PORT}..."

wait_for_postgres() {
    if command -v nc >/dev/null 2>&1; then
        until nc -z "$DB_HOST" "$DB_PORT" 2>/dev/null; do
            echo "Waiting for PostgreSQL..."
            sleep 2
        done
    else
        echo "nc not available, waiting 10s for PostgreSQL..."
        sleep 10
    fi
}

wait_for_postgres

echo "PostgreSQL is ready"

sed -i "s|@@DB_HOST@@|${DB_HOST}|g" /etc/raddb/mods-enabled/sql
sed -i "s|@@DB_PORT@@|${DB_PORT}|g" /etc/raddb/mods-enabled/sql
sed -i "s|@@DB_USER@@|${DB_USER}|g" /etc/raddb/mods-enabled/sql
sed -i "s|@@DB_PASSWORD@@|${DB_PASSWORD}|g" /etc/raddb/mods-enabled/sql
sed -i "s|@@DB_NAME@@|${DB_NAME}|g" /etc/raddb/mods-enabled/sql
sed -i "s|@@RADIUS_SECRET@@|${RADIUS_SECRET}|g" /etc/raddb/clients.conf

exec freeradius -f -l stdout
