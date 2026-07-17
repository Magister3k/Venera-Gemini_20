# Dockerfile для приложения Venera
# Обратите внимание: поскольку код содержит специфичные для Windows системные вызовы
# (sys/windows/svc, WinAPI, ETW), данный Dockerfile использует Linux-окружение только
# если в коде предусмотрены заглушки (build tags) для Linux.
# Для 100% совместимости с WinAPI этот Dockerfile служит шаблоном архитектуры (п.31 ТЗ).

# Этап 1: Сборка фронтенда (React-UI)
FROM node:20-alpine AS frontend-builder
WORKDIR /app/react-ui
COPY react-ui/package*.json ./
RUN npm install
COPY react-ui/ .
RUN npm run build

# Этап 2: Сборка Go приложения
FROM golang:1.21-alpine AS backend-builder
WORKDIR /app

# Устанавливаем необходимые зависимости для сборки (если требуется cgo)
RUN apk add --no-cache gcc musl-dev

# Копируем go.mod и go.sum
COPY go.mod go.sum ./
RUN go mod download

# Копируем весь проект
COPY . .

# Сборка приложения
# Если приложение содержит специфичный для Windows код без условной компиляции,
# сборка под linux упадет. В таком случае рекомендуется собирать под GOOS=windows
# и запускать в Windows Server Core контейнере.
# Но для шаблона используем стандартную сборку:
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o venera_app main.go

# Этап 3: Финальный легковесный образ
FROM alpine:latest
WORKDIR /app

# Устанавливаем tshark (зависимость для работы приложения)
RUN apk add --no-cache tshark tzdata

# Копируем бинарник
COPY --from=backend-builder /app/venera_app .

# Копируем статику фронтенда
COPY --from=frontend-builder /app/react-ui/dst ./react-ui/dst

# Создаем необходимые директории
RUN mkdir -p Logs settings backups

# Открываем порт
EXPOSE 8080

# Точка входа
CMD ["./venera_app"]
