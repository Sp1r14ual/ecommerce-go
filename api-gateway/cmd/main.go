package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"

	pb "github.com/Sp1r14ual/ecommerce-go/proto/auth"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Структуры для парсинга входящего JSON
type AuthRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func main() {
	// 1. Создаем соединение с gRPC сервером (наш Auth Service)
	// В Go > 1.21 используется grpc.NewClient вместо устаревшего grpc.Dial
	authAddr := os.Getenv("AUTH_SERVICE_ADDR")
	if authAddr == "" {
		authAddr = "localhost:50051"
	}
	conn, err := grpc.NewClient(authAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	// 2. Инициализируем gRPC клиент (код сгенерирован автоматически в grpc.pb.go)
	authClient := pb.NewAuthServiceClient(conn)

	// 3. Создаем HTTP роутер (маршрутизатор)
	mux := http.NewServeMux()

	// 4. Эндпоинт для регистрации
	mux.HandleFunc("POST /api/register", func(w http.ResponseWriter, r *http.Request) {
		var req AuthRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Вызываем метод Register по gRPC
		grpcResp, err := authClient.Register(context.Background(), &pb.RegisterRequest{
			Email:    req.Email,
			Password: req.Password,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Отвечаем клиенту JSON'ом
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"user_id": grpcResp.GetUserId()})
	})

	// 5. Эндпоинт для логина
	mux.HandleFunc("POST /api/login", func(w http.ResponseWriter, r *http.Request) {
		var req AuthRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Вызываем метод Login по gRPC
		grpcResp, err := authClient.Login(context.Background(), &pb.LoginRequest{
			Email:    req.Email,
			Password: req.Password,
		})
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Отдаем токен в формате JSON
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"access_token": grpcResp.GetAccessToken()})
	})

	// 6. Запускаем HTTP сервер
	log.Println("API Gateway is running on http://localhost:8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatalf("failed to serve HTTP: %v", err)
	}
}
