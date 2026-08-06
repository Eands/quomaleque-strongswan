# VPN Management — strongSwan + FreeRADIUS

Fully containerized VPN gateway with web management panel.

## Stack

- **strongSwan 5.9+** — IKEv2 VPN server with EAP-MSCHAPv2 + RADIUS auth
- **FreeRADIUS 3.0+** — RADIUS server backed by PostgreSQL
- **Go + Gin** — Web management panel
- **PostgreSQL 15** — User storage, accounting logs, app settings

## Quick Start

```bash
# 1. Configure environment
cp .env.example .env
# Edit .env — set passwords and secrets

# 2. Build and start
make up

# 3. Access web panel
# https://localhost:443
# Login: admin / change_me_admin_password (from .env)
```

## Makefile Commands

| Command | Description |
|---------|-------------|
| `make up` | Build and start all services |
| `make down` | Stop all services |
| `make build` | Rebuild all images (no cache) |
| `make logs` | Tail all service logs |
| `make restart` | Restart all services |
| `make clean` | Stop services and remove volumes/data |

## Environment Variables

See `.env.example` for all available variables.

| Variable | Default | Description |
|----------|---------|-------------|
| `DB_HOST` | postgres | Database host |
| `DB_PORT` | 5432 | Database port |
| `DB_USER` | radius | Database user |
| `DB_PASSWORD` | — | Database password |
| `DB_NAME` | radius | Database name |
| `RADIUS_SECRET` | — | Shared secret for strongSwan↔FreeRADIUS |
| `STRONGSWAN_VICI_ADDR` | strongswan:4502 | VICI TCP address |
| `STRONGSWAN_IPSEC_POOL_START` | 10.10.0.1 | VPN IP pool start |
| `STRONGSWAN_IPSEC_POOL_END` | 10.10.0.254 | VPN IP pool end |
| `STRONGSWAN_DNS` | 8.8.8.8,1.1.1.1 | DNS servers for VPN clients |
| `LOG_RETENTION_DAYS` | 90 | Accounting log retention |
| `APP_USER` | admin | Web panel username |
| `APP_PASSWORD` | — | Web panel password |

## HTTPS Certificates

By default, a self-signed certificate is generated at startup. To use Let's Encrypt:

1. Obtain certificates (e.g., via certbot)
2. Set `SSL_CERT_PATH=/certs/fullchain.pem` and `SSL_KEY_PATH=/certs/privkey.pem`
3. Mount the certificate files to `/certs/` in the webapp container

## Adding VPN Users

1. Log into the web panel
2. Go to **Users** → **Add User**
3. Fill in username, password, optional group and max connections
4. Configure VPN client with:
   - Server: your server IP/hostname
   - Auth: EAP-MSCHAPv2 (username/password)
   - Certificate: use the CA certificate from the strongSwan volume

### Export CA Certificate

```bash
docker cp strongswan:/etc/ipsec.d/cacerts/ca-cert.pem ./ca-cert.pem
```

## Architecture

```
                    ┌─────────────┐              ┌─────────────┐   ┌─────────────┐
    UDP 500/4500 ──▶│ strongSwan  │◀── RADIUS ──▶│ FreeRADIUS  │──▶│ PostgreSQL  │
                    │  (ipsec)    │              │  (radiusd)  │   │             │
                    └──────┬──────┘              └─────────────┘   └─────────────┘
                           │ VICI TCP :4502
                           ▼
                    ┌─────────────┐   ┌─────────────┐
      HTTPS :443 ──▶│  Web App    │──▶│ PostgreSQL  │
      HTTP :80  ──▶ │  (Go/Gin)   │   └─────────────┘
                    └─────────────┘
```
