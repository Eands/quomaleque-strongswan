# VPN Management

IKEv2 VPN на базе StrongSwan и ToughRADIUS.

## Структура

```
├── docker-compose.yml              # dev (по умолчанию) + prod (--profile prod)
├── strongswan/
│   ├── Dockerfile                  # Ubuntu + strongswan
│   ├── startup.sh                  # старт charon + загрузка конфига
│   ├── config/
│   │   ├── strongswan.conf         # eap-radius → 127.0.0.1:1812
│   │   └── swanctl/
│   │       ├── swanctl.conf        # IKEv2 connection, IP pool
│   │       ├── x509/               # сертификаты (генерятся generate_certs.sh)
│   │       ├── x509ca/
│   │       └── private/
│   └── scripts/
│       └── generate_certs.sh       # самоподписанные сертификаты
├── toughradius/
│   ├── config/
│   │   └── toughradius.yml         # конфиг RADIUS-сервера
│   └── data/                       # SQLite БД
├── nginx/
│   └── nginx.conf                  # HTTPS proxy → toughradius:1816
└── certbot/
    └── deploy.sh                   # копирование LE-сертификатов в certs volume
```

## Запуск

### Dev (самоподписанные сертификаты, локальная отладка)

```bash
# Установить strongswan-pki (для генерации сертификатов)
sudo apt-get install -y strongswan-pki openssl

./strongswan/scripts/generate_certs.sh
docker compose up -d --build

# Инициализация БД ToughRADIUS (одноразово)
docker exec toughradius toughradius -initdb -c /etc/toughradius/toughradius.yml
```

**Доступ:**
| Ресурс | Адрес |
|--------|-------|
| ToughRADIUS Admin | `http://<IP_VM>:1816` (admin / toughradius) |
| VPN-шлюз | IKEv2 → `<IP_VM>` |

После первого входа сменить пароль admin и **добавить NAS**:
- IP: `127.0.0.1`
- Secret: `HpE98gAFA4OaJaHYU46M` (как в `strongswan.conf`)

Затем создать VPN-пользователей через Admin UI.

### Prod (Let's Encrypt + HTTPS)

```bash
export DOMAIN=vpn.example.com

# Первый запуск — получить сертификат
docker compose --profile prod run --rm certbot certonly \
  --webroot --webroot-path /var/www/certbot \
  --email admin@example.com --agree-tos -d $DOMAIN

# Запустить всё
docker compose --profile prod up -d --build

# Инициализация БД (одноразово)
docker exec toughradius toughradius -initdb -c /etc/toughradius/toughradius.yml
```

**Доступ:**
| Ресурс | Адрес |
|--------|-------|
| ToughRADIUS Admin | `https://vpn.example.com` |
| VPN-шлюз | IKEv2 → `vpn.example.com` |

Сертификаты обновляются автоматически каждые 7 дней (certbot container).

## Сертификаты

- **Dev:** самоподписанные, генерируются `generate_certs.sh`, монтируются в strongswan
- **Prod:** Let's Encrypt через certbot, общий `certs` volume для strongswan-prod и nginx

## Настройка VPN-клиента

1. **Dev:** импортировать `strongswan/config/swanctl/x509ca/caCert.pem` (или `clientCert.p12`)
2. **Prod:** ничего импортировать не нужно (LE-сертификат доверенный)
3. Создать IKEv2-подключение: сервер = IP/домен, аутентификация EAP (логин/пароль из Admin UI)

## Отладка

```bash
# Логи strongswan
docker logs -f strongswan

# Логи ToughRADIUS
docker logs -f toughradius

# Проверить RADIUS (после добавления NAS)
radtest testuser testpass 127.0.0.1 1812 HpE98gAFA4OaJaHYU46M
```
