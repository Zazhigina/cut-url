# cut-url

Сервис сокращения ссылок. Хранилище — в памяти или PostgreSQL, выбирается при запуске.

## Запуск

Сервис поднимается на `http://localhost:8080`.

**Docker + PostgreSQL** — база и схема создаются сами, настраивать нечего:

```sh
docker compose up --build
```

Остановить — `docker compose down`, вместе с данными — `docker compose down -v`.

**Без Docker, только память** — ни Docker, ни PostgreSQL не нужны:

```sh
go run ./cmd/server -storage=memory
```

Хранилище по умолчанию и так `memory`, флаг можно не указывать — достаточно
`go run ./cmd/server`.

## Параметры запуска

Задаются флагом или переменной окружения; флаг приоритетнее.

| Флаг | Переменная | По умолчанию | Что задаёт |
|---|---|---|---|
| `-storage` | `STORAGE_TYPE` | `memory` | Хранилище: `memory` или `postgres` |
| `-port` | `PORT` | `8080` | Порт сервера |

Подключение к PostgreSQL задаётся только переменными окружения: `DB_HOST`,
`DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`, `DB_SSLMODE`.
