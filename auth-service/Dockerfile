# Этап сборки (builder)
FROM golang:alpine AS builder

WORKDIR /app
# Копируем go.mod и go.sum из корня
COPY go.mod go.sum ./
RUN go mod download

# Копируем весь исходный код
COPY . .
# Собираем бинарник именно для auth-service
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./auth-service/cmd/main.go

# Финальный этап (очень легкий образ)
FROM alpine:latest
WORKDIR /root/
# Копируем готовый бинарник из builder
COPY --from=builder /app/main .

EXPOSE 50051
CMD ["./main"]