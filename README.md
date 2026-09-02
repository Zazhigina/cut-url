# cut-url

Сервис сокращения ссылок. Хранилище — в памяти или PostgreSQL, выбирается при запуске.

## Запуск

Сервис поднимается на `http://localhost:8080`.

**Docker + PostgreSQL** — база и схема создаются сами, настраивать нечего:

```sh
docker compose up --build
```

Остановить — `docker compose down`, вместе с данными — `docker compose down -v`.

**Docker + память** — база не нужна:

```sh
docker build -t cut-url .
docker run --rm -p 8080:8080 cut-url
```

## API

Создать короткую ссылку:

```sh
curl -X POST http://localhost:8080/shorten \
  -H "Content-Type: application/json" \
  -d "{\"url\":\"https://example.com/very/long/path\"}"
```

Перейти по ней:

```sh
curl -i http://localhost:8080/hf6Vs9rTiT
```


