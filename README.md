# Disciplinary Committee

[Docker compose]


[Traefik dashboard]
https://dashboard.docker.localhost/

Как это работает:

    Клиент отправляет запрос: https://spbgu.local (Порт 443).
    Traefik принимает его, расшифровывает с помощью сертификата.
    Traefik видит, что запрошен discom.spbgu.local, и отправляет обычный HTTP запрос внутри Docker-сети на IP вашего контейнера 172.22.0.21:80.
    Приложение отвечает по HTTP, Traefik упаковывает ответ в HTTPS и отдает клиенту.

[Portainer]
https://localhost:9443
admin_portainer:L)y)0988N:3A

[MacOS]
```bash
    find . -type f -name "db.sh" -exec chmod +x {} +
```

[front with build]
# Многоэтапная сборка
#FROM node:18-alpine AS builder
#WORKDIR /app

#COPY package*.json ./
#RUN npm ci --only=production || true

#COPY . .
#RUN npm run build || true  # Если есть скрипт сборки

Test ci/cd №6
