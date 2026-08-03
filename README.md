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

Веб-интерфейс: `http://<IP_VM>:8080/login`
Логин: `admin`, пароль: `admin` (создаётся автоматически при первом запуске)

### Продакшен (Let's Encrypt + HTTPS)

```bash
# 1. Получить сертификат
export DOMAIN=vpn.example.com
docker compose -f docker-compose.yml -f docker-compose.prod.yml run --rm certbot \
  certonly --webroot --webroot-path /var/www/certbot \
    --email admin@example.com --agree-tos -d $DOMAIN

# 2. Запустить
docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d --build
```

Веб-интерфейс: `https://vpn.example.com`

### Настройка VPN-клиента

1. Импортировать CA-сертификат на устройство:
   - **Локально:** `strongswan/config/swanctl/clientCert.p12`
   - **Продакшен:** сертификат доверенный автоматически (Let's Encrypt)

2. Создать IKEv2-подключение:
   - Сервер: домен или IP сервера
   - Remote ID: домен сервера
   - Аутентификация: EAP (логин / пароль из веб-интерфейса)

### Отладка

```bash
docker compose logs -f radius strongswan

docker exec vpn-radius sqlite3 /data/users.db "SELECT * FROM users;"
docker exec vpn-radius sqlite3 /data/users.db "SELECT * FROM radius_logs ORDER BY timestamp DESC LIMIT 10;"
docker exec vpn-radius sqlite3 /data/users.db "SELECT * FROM schema_migrations;"

docker exec strongswan swanctl --list-sas
```
