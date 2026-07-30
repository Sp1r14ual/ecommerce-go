FROM golang:alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Собираем goods-service
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./goods-service/cmd/main.go

FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/main .

EXPOSE 50052
CMD ["./main"]