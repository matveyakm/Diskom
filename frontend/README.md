# Frontend

## Production (через nginx)

```bash
docker compose up -d frontend-app
```

При изменении кода — полная пересборка:
```bash
docker compose build frontend-app
docker compose up -d frontend-app
```

## Dev-режим (Vite с HMR)

Изменения в коде сразу видны в браузере без пересборки контейнера.

```bash
docker compose rm -f -s frontend-app
docker compose --profile dev up -d frontend-dev
```

Откройте `https://discom.spbgu.localhost`

## Вернуть production

```bash
# 1. Удалить dev-контейнер
docker compose rm -f -s frontend-dev

# 2. Запустить production-фронт
docker compose up -d frontend-app
```
