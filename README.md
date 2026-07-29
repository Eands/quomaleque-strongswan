# VPN Management

Система управления IKEv2 VPN на базе StrongSwan, Go RADIUS-сервера и веб-интерфейса.

## Архитектура

| Компонент | Технология | Порт | Описание |
|-----------|-----------|------|---------|
| **strongswan** | Docker (`jesusdf/docker-strongswan`) | 500/udp, 4500/udp | IKEv2 VPN-шлюз, аутентификация через EAP-RADIUS |
| **radius** | Go (`layeh.com/radius`, Gin) | 1812/udp, 1813/udp, 8080 | RADIUS-сервер (auth + accounting), веб-интерфейс управления |
| **VICI** | Unix socket (`/tmp/vici/charon.vici`) | — | Протокол управления StrongSwan из Go-приложения |

```
Клиент ──(IKEv2)──> strongswan ──(RADIUS)──> radius ──(VICI)──> strongswan
                                               │
                                               ├── SQLite (users.db)
                                               └── Web UI (:8080)
```

## Структура проекта

```
├── docker-compose.yml
├── strongswan/
│   ├── config/
│   │   ├── strongswan.conf      # Конфиг charon (VICI, EAP-RADIUS)
│   │   └── swanctl/
│   │       └── swanctl.conf     # IKEv2 connection + IP pool
│   └── scripts/
│       └── generate_certs.sh    # Генерация сертификатов
└── radius/
    ├── Dockerfile               # Multi-stage Go build
    ├── main/
    │   └── main.go               # Точка входа: RADIUS + Web
    └── internal/
        ├── db/
        │   ├── database.go       # SQLite слой
        │   ├── migrations.go     # Embed-миграции
        │   └── migrations/
        │       └── 001_initial.sql
        ├── handlers/
        │   └── handlers.go       # HTTP-обработчики (Gin)
        ├── vici/
        │   └── manager.go        # VICI-клиент (govici)
        └── web/
            ├── static/
            │   └── style.css
            └── templates/        # HTML-шаблоны
```

## Быстрый старт

### 1. Генерация сертификатов

```bash
cd strongswan/scripts
chmod +x generate_certs.sh
./generate_certs.sh
```

Сертификаты создадутся в `strongswan/config/swanctl/x509ca/`, `x509/`, `private/` и `clientCert.p12`.

### 2. Запуск

```bash
# Сборка и запуск в фоне
docker compose up -d --build

# Просмотр логов
docker compose logs -f radius strongswan
```

Ожидаемые логи radius:
```
RADIUS authentication server starting on :1812
RADIUS accounting server starting on :1813
Web server starting on :8080
Applied migration: 001_initial.sql
```

Возможное предупреждение при первом запуске:
```
Warning: Failed to connect to VICI: ...
```
Это нормально — strongswan ещё создаёт VICI-сокет.

### 3. Создание первого пользователя

Веб-интерфейс требует авторизации, поэтому первый пользователь создаётся напрямую в БД:

```bash
# Сгенерировать bcrypt-хеш (go/bcrypt)
docker exec vpn-radius sh -c '
echo "import (
  \"fmt\"
  \"golang.org/x/crypto/bcrypt\"
)
func main() {
  hash, _ := bcrypt.GenerateFromPassword([]byte(\"admin\"), bcrypt.DefaultCost)
  fmt.Println(string(hash))
}" > /tmp/hash.go
'
```

Или проще — использовать Python:

```bash
HASH=$(python3 -c 'import bcrypt; print(bcrypt.hashpw(b"admin", bcrypt.gensalt()).decode())')
```

Или создать через скрипт внутри контейнера:

```bash
docker exec vpn-radius sqlite3 /data/users.db \
  "INSERT INTO users (username, password_hash) VALUES ('admin',
  '\$2a\$10\$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy');"
```

После этого можно зайти в веб-интерфейс:
```
http://localhost:8080/login
```

Логин: `admin`, пароль: `admin`

### 4. Добавление пользователей через веб-интерфейс

После входа:
- **Users → Create User** — добавить VPN-пользователей
- Пароль хешируется bcrypt автоматически
- Созданные пользователи сразу доступны для VPN-подключения

## Проверка работы

### Проверка RADIUS

```bash
# Установить утилиту (macOS)
brew install freeradius-utils

# Протестировать аутентификацию
radtest testuser testpass 127.0.0.1 1812 HpE98gAFA4OaJaHYU46M
```

Ожидаемый ответ: `Access-Accept`

### Проверка VICI (управление strongswan)

```bash
# Сокет должен быть доступен через shared volume
docker exec vpn-radius ls -la /tmp/vici/
```

В веб-интерфейсе: **VPN** — показывает подключения и активные SA.

### Проверка сессий и логов

- **Sessions** — активные/завершённые VPN-сессии (автоматически из RADIUS-аккаунтинга)
- **Logs** — история RADIUS-запросов (Access-Request, Accounting-Request)
- **Stats** — общая статистика

## Настройка VPN-клиента

### macOS / iOS (встроенный клиент)

1. Импортировать `clientCert.p12` в Keychain
2. Создать VPN-подключение типа IKEv2:
   - Сервер: `quomaleque.ru` (или IP сервера)
   - Удалённый ID: `quomaleque.ru`
   - Аутентификация: EAP (имя пользователя / пароль)
   - Имя пользователя и пароль — из веб-интерфейса

### Windows

1. Импортировать `strongswan/config/swanctl/x509ca/caCert.pem` в Trusted Root CA
2. Создать VPN-подключение IKEv2
3. Параметры как для macOS

### Linux (strongswan client)

```bash
# /etc/ipsec.conf
conn vpn
    keyexchange=ikev2
    leftauth=eap
    right=quomaleque.ru
    rightid=quomaleque.ru
    rightauth=pubkey
    rightsubnet=0.0.0.0/0
    auto=add

# /etc/ipsec.secrets
username : EAP "password"
```

## Переменные окружения

| Переменная | По умолчанию | Где используется |
|-----------|-------------|-----------------|
| `RADIUS_SECRET` | `HpE98gAFA4OaJaHYU46M` | RADIUS-сервер — должен совпадать с `strongswan.conf` |
| `VICI_SOCKET` | `/var/run/charon.vici` | Путь к VICI Unix-сокету |
| `DB_PATH` | `/data/users.db` | Путь к SQLite БД |

## Отладка

```bash
# Все логи
docker compose logs -f

# Логи только radius
docker compose logs -f radius

# Проверить БД
docker exec vpn-radius sqlite3 /data/users.db ".tables"
docker exec vpn-radius sqlite3 /data/users.db "SELECT * FROM users;"
docker exec vpn-radius sqlite3 /data/users.db "SELECT * FROM schema_migrations;"

# Проверить логи RADIUS
docker exec vpn-radius sqlite3 /data/users.db "SELECT * FROM radius_logs ORDER BY timestamp DESC LIMIT 10;"

# Проверить VICI
docker exec strongswan swanctl --list-sas

# Зайти внутрь контейнера
docker exec -it vpn-radius sh
docker exec -it strongswan sh
```

## Добавление новых миграций

1. Создать файл `radius/internal/db/migrations/002_описание.sql`
2. При следующем запуске миграция применится автоматически
3. Повторное применение пропускается (отслеживается в `schema_migrations`)

## Продакшен-рекомендации

- Сменить `RADIUS_SECRET` (совпадает в `.env` и `strongswan.conf`)
- Сменить секретный ключ сессий в `main.go` (`vpn-radius-session-secret-key`)
- Настроить реальный домен (`quomaleque.ru` → ваш домен в `generate_certs.sh` и `swanctl.conf`)
- Добавить `nginx` перед веб-интерфейсом с HTTPS
- Использовать внешнюю БД (PostgreSQL) вместо SQLite при высокой нагрузке
