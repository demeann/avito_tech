# PR Reviewer Assignment Service — Test Task (Avito, Fall 2025)

Микросервис для автоматического назначения ревьюеров на Pull Request'ы.

## Стек

- Go 1.23
- PostgreSQL 16
- pgx/v5 (драйвер PostgreSQL)
- Docker / Docker Compose
- HTTP API соответствует выданной OpenAPI-спецификации (см. `openapi.yml`)

## Быстрый старт

### Запуск через Docker Compose

Требуется установленный Docker и Docker Compose.

```bash
docker-compose up --build
```

Или используя Makefile:

```bash
make docker-up
```

После запуска сервис доступен на:

- `http://localhost:8080/health` — health-check
- остальные эндпоинты — согласно `openapi.yml`

База данных поднимается вместе с сервисом, миграции выполняются автоматически при старте приложения (через `AutoMigrate`).

### Локальный запуск

Для локального запуска требуется запущенная база данных PostgreSQL.

```bash
make run-local
```

Или вручную:

```bash
docker-compose up -d db
export DATABASE_URL="postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"
export APP_PORT="8080"
make run
```

## Конфигурация

Все параметры конфигурации настраиваются через переменные окружения:

- `APP_PORT` — порт для HTTP сервера (по умолчанию: `8080`)
- `DATABASE_URL` — строка подключения к PostgreSQL (по умолчанию: `postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable`)

Для Docker Compose также доступны:

- `POSTGRES_USER` — пользователь PostgreSQL (по умолчанию: `postgres`)
- `POSTGRES_PASSWORD` — пароль PostgreSQL (по умолчанию: `postgres`)
- `POSTGRES_DB` — имя базы данных (по умолчанию: `postgres`)
- `POSTGRES_PORT` — порт PostgreSQL (по умолчанию: `5432`)

Пример настройки через `.env` файл:

```bash
POSTGRES_USER=myuser
POSTGRES_PASSWORD=mypassword
POSTGRES_DB=mydb
APP_PORT=3000
DATABASE_URL=postgres://myuser:mypassword@db:5432/mydb?sslmode=disable
```

## Архитектура

Проект разделён на уровни:

- **`internal/domain`** — доменные сущности (Team, User, PullRequest и т.д.).
- **`internal/repository/postgres`** — реализация слоя доступа к данным (PostgreSQL через pgx/v5).
- **`internal/service`** — бизнес-логика и доменные правила (назначение ревьюверов, перенос, merge и т.п.).
- **`internal/httpserver`** — HTTP-слой, маппинг JSON <-> доменные модели, кодов ошибок в формат OpenAPI.
- **`cmd/server`** — точка входа, wiring зависимостей.

### Основные доменные правила

1. **Создание команды `/team/add`**
   - Если команда с таким `team_name` уже существует — возвращается ошибка `400 TEAM_EXISTS`.
   - Иначе создаётся команда и её участники.
   - Пользователи создаются/обновляются (upsert) с привязкой к новой команде.

2. **Получение команды `/team/get`**
   - Возвращает команду и всех её участников.
   - Если команда не найдена — `404 NOT_FOUND`.

3. **Установка активности пользователя `/users/setIsActive`**
   - Обновляет флаг `is_active`.
   - Возвращает пользователя с `team_name`.
   - Если пользователь не найден — `404 NOT_FOUND`.

4. **Создание PR `/pullRequest/create`**
   - Если PR с таким `pull_request_id` уже существует — `409 PR_EXISTS`.
   - Проверяется существование автора и его команды (`NOT_FOUND`, если нет).
   - Автоматически назначаются до **2** активных ревьюверов из команды автора, исключая автора.
   - Если доступных кандидатов меньше двух — назначается 0/1 в соответствии с количеством.

5. **Merge PR `/pullRequest/merge`**
   - Идемпотентная операция:
     - Если PR в статусе `OPEN` — переводится в `MERGED`, выставляется `mergedAt` (если было `NULL`).
     - Если уже `MERGED` — просто возвращается текущее состояние (без ошибки).
   - Если PR не найден — `404 NOT_FOUND`.

6. **Переназначение ревьювера `/pullRequest/reassign`**
   - Проверяется, что PR существует и не в статусе `MERGED`:
     - Если `MERGED` — `409 PR_MERGED`.
   - Проверяется, что `old_user_id` действительно назначен ревьювером:
     - Если нет — `409 NOT_ASSIGNED`.
   - Ищется команда старого ревьювера, выбирается **случайный** активный участник этой команды:
     - Исключаются: сам `old_user_id` и остальные текущие ревьюверы PR (чтобы избежать дубликатов).
     - Если подходящих кандидатов нет — `409 NO_CANDIDATE`.
   - В режиме транзакции старый ревьювер заменяется новым.

7. **Получение PR'ов пользователя `/users/getReview`**
   - Возвращает все PR, где пользователь назначен ревьювером (короткое представление `PullRequestShort`).
   - Если пользователь не найден — `404 NOT_FOUND`.

### Структура БД

Миграции выполняются кодом (см. `internal/repository/postgres/migrations.go`):

- `teams` — команды (`id`, `name UNIQUE`)
- `users` — пользователи (`user_id PK`, `username`, `team_id FK`, `is_active`)
- `pull_requests` — PR (`pr_id PK`, `pr_name`, `author_id FK`, `status`, `created_at`, `merged_at`)
- `pull_request_reviewers` — связь PR и ревьюверов (`pr_id FK`, `reviewer_id FK`, PK по двум полям)

## Makefile

Доступные команды:

```bash
make build          # сборка бинарника
make run            # локальный запуск (требует DATABASE_URL в окружении)
make run-local      # запуск с автоматическим поднятием БД в Docker
make docker-up      # запуск всех сервисов в Docker
make docker-down    # остановка всех сервисов
make docker-logs    # просмотр логов приложения
make docker-restart # перезапуск приложения
make test           # запуск тестового скрипта API
```

## Допущения и пояснения

1. **OpenAPI несовершенства / оговорки**
   - В примере для `/pullRequest/reassign` поле называется `old_reviewer_id`, но в схеме — `old_user_id`.  
     В коде используется **`old_user_id`**, как в схеме.
   - В некоторых местах OpenAPI не задаёт точный формат ошибок — принят единый формат:
     ```json
     {
       "error": {
         "code": "NOT_FOUND",
         "message": "..."
       }
     }
     ```

2. **Статусы PR**
   - Хранятся в БД в текстовом виде (`OPEN` / `MERGED`), в домене — `PRStatus`.

3. **Выбор случайных ревьюверов**
   - Используется `math/rand` с seed по времени запуска.
   - Для небольших объёмов данных и RPS этого достаточно.

4. **Иммутабельность ревьюверов после MERGE**
   - Любая попытка вызова `/pullRequest/reassign` для PR со статусом `MERGED` приводит к `409 PR_MERGED`, что соответствует требованиям.

5. **Миграции**
   - Используется простой код миграций при старте вместо отдельного мигратора/миграций в файлах.  
     Для промышленного использования подошли бы инструменты вроде `goose`/`migrate`, но в рамках тестового задания это упрощает запуск командой `docker-compose up`.

6. **База данных**
   - Используется драйвер `pgx/v5` для работы с PostgreSQL.
   - Подключение через connection pool (`pgxpool.Pool`).

## Пример последовательности запросов

1. Создать команду:

```bash
curl -X POST http://localhost:8080/team/add \
  -H "Content-Type: application/json" \
  -d '{
    "team_name": "backend",
    "members": [
      {"user_id": "u1", "username": "Alice", "is_active": true},
      {"user_id": "u2", "username": "Bob", "is_active": true},
      {"user_id": "u3", "username": "Charlie", "is_active": true}
    ]
  }'
```

2. Создать PR:

```bash
curl -X POST http://localhost:8080/pullRequest/create \
  -H "Content-Type: application/json" \
  -d '{
    "pull_request_id": "pr-1001",
    "pull_request_name": "Add search",
    "author_id": "u1"
  }'
```

3. Переназначить ревьювера:

```bash
curl -X POST http://localhost:8080/pullRequest/reassign \
  -H "Content-Type: application/json" \
  -d '{
    "pull_request_id": "pr-1001",
    "old_user_id": "u2"
  }'
```

4. Merge PR (операция идемпотентна):

```bash
curl -X POST http://localhost:8080/pullRequest/merge \
  -H "Content-Type: application/json" \
  -d '{ "pull_request_id": "pr-1001" }'
```

5. Получить PR'ы, где пользователь ревьювер:

```bash
curl "http://localhost:8080/users/getReview?user_id=u2"
```

## Тестирование

Для автоматического тестирования API доступен скрипт:

```bash
./test_api.sh
```

Или через Makefile:

```bash
make test
```

## Линтер / качество кода

В реальном проекте использовал бы:

- `golangci-lint` с набором базовых линтеров (`govet`, `staticcheck`, `errcheck`, `gocyclo` и др.).
- Настройки разместил бы в `.golangci.yml` в корне.

В рамках тестового задания конфигурация линтера описана текстом, чтобы не перегружать репозиторий.

---
