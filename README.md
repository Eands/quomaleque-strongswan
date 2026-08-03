# VPN Management

IKEv2 VPN на базе StrongSwan, Go RADIUS-сервера и веб-интерфейса.

## Структура

```
├── docker-compose.yml           # Базовый (локальный запуск)
├── docker-compose.prod.yml      # Продакшен (nginx + Let's Encrypt)
├── nginx/
│   └── nginx.conf               # HTTPS proxy
├── certbot/
│   └── deploy.sh                # Копирование LE-сертификатов → swanctl
├── strongswan/
│   ├── Dockerfile                # Свой образ (Ubuntu + strongswan)
│   ├── config/
│   │   ├── strongswan.conf      # charon: VICI, EAP-RADIUS
│   │   └── swanctl/
│   │       └── swanctl.conf     # IKEv2 connection, IP pool
│   └── scripts/
│       └── generate_certs.sh    # Самоподписанные сертификаты
└── radius/
    ├── Dockerfile
    ├── main/main.go             # Точка входа: RADIUS + Web
    └── internal/
        ├── db/                  # SQLite, миграции, seed admin
        ├── handlers/            # HTTP-обработчики (Gin)
        ├── vici/                # VICI-клиент (govici)
        └── web/
            ├── static/style.css
            └── templates/       # HTML-шаблоны
```

## Запуск

### Локально (самоподписанные сертификаты)

```bash
# Установить strongswan-pki (нужен для генерации сертификатов)
sudo apt-get install -y strongswan-pki openssl

./strongswan/scripts/generate_certs.sh
docker compose up -d --build
```

### Let's Encrypt + HTTPS (not tested yet)

```bash
# 1. Получить сертификат
export DOMAIN=vpn.example.com
docker compose -f docker-compose.yml -f docker-compose.prod.yml run --rm certbot \
  certonly --webroot --webroot-path /var/www/certbot \
    --email admin@example.com --agree-tos -d $DOMAIN

# 2. Запустить
docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d --build
```

### Debug

```bash
docker compose --profile debug up -d --build radius-debug
```

### TODO