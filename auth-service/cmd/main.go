package main

import (
	"context"
	"log"
	"net"

	"github.com/Sp1r14ual/ecommerce-go/auth-service/internal/repository"
	"github.com/Sp1r14ual/ecommerce-go/auth-service/internal/server"
	"github.com/Sp1r14ual/ecommerce-go/auth-service/internal/service"
	pb "github.com/Sp1r14ual/ecommerce-go/proto/auth"

	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	// 1. Строка подключения к нашему Postgres из docker-compose
	connString := "postgres://auth_user:auth_password@localhost:5432/auth_db?sslmode=disable"

	ctx := context.Background()
	dbPool, err := pgxpool.New(ctx, connString)
	if err != nil {
		log.Fatalf("Unable to create connection pool: %v", err)
	}
	defer dbPool.Close()

	// 1.1 Создаем таблицу (В реальных проектах используют миграции, но для начала сойдет так)
	_, err = dbPool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS users (
			id SERIAL PRIMARY KEY,
			email VARCHAR(255) UNIQUE NOT NULL,
			password_hash VARCHAR(255) NOT NULL
		);
	`)
	if err != nil {
		log.Fatalf("Failed to create table: %v", err)
	}

	// 2. Инициализируем слои по порядку (Dependency Injection)
	repo := repository.NewAuthRepo(dbPool)

	// В реальном проекте секрет берется из переменных окружения (os.Getenv)
	svc := service.NewAuthService(repo, "my-super-secret-key-for-jwt")

	grpcHandler := server.NewAuthServer(svc)

	// 3. Запускаем gRPC сервер
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()

	// Регистрируем наш обработчик в сервере
	pb.RegisterAuthServiceServer(s, grpcHandler)

	reflection.Register(s)

	log.Println("Auth gRPC Service is running on port 50051...")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
