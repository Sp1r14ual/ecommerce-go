package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"

	authPb "github.com/Sp1r14ual/ecommerce-go/proto/auth"   // Алиас для пакета auth
	goodsPb "github.com/Sp1r14ual/ecommerce-go/proto/goods" // Алиас для пакета goods

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type AuthRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type CreateProductRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Price       int64  `json:"price"`
}

func main() {
	// --- Подключение к Auth Service ---
	authAddr := os.Getenv("AUTH_SERVICE_ADDR")
	if authAddr == "" {
		authAddr = "localhost:50051"
	}
	authConn, err := grpc.NewClient(authAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("failed to connect to auth: %v", err)
	}
	defer authConn.Close()
	authClient := authPb.NewAuthServiceClient(authConn)

	// --- Подключение к Goods Service ---
	goodsAddr := os.Getenv("GOODS_SERVICE_ADDR")
	if goodsAddr == "" {
		goodsAddr = "localhost:50052"
	}
	goodsConn, err := grpc.NewClient(goodsAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("failed to connect to goods: %v", err)
	}
	defer goodsConn.Close()
	goodsClient := goodsPb.NewGoodsServiceClient(goodsConn)

	// --- Настройка Роутера ---
	mux := http.NewServeMux()

	// 1. Ручки Авторизации
	mux.HandleFunc("POST /api/register", func(w http.ResponseWriter, r *http.Request) {
		var req AuthRequest
		json.NewDecoder(r.Body).Decode(&req)

		resp, err := authClient.Register(context.Background(), &authPb.RegisterRequest{
			Email: req.Email, Password: req.Password,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"user_id": resp.GetUserId()})
	})

	mux.HandleFunc("POST /api/login", func(w http.ResponseWriter, r *http.Request) {
		var req AuthRequest
		json.NewDecoder(r.Body).Decode(&req)

		resp, err := authClient.Login(context.Background(), &authPb.LoginRequest{
			Email: req.Email, Password: req.Password,
		})
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"access_token": resp.GetAccessToken()})
	})

	// 2. Ручки Товаров
	mux.HandleFunc("POST /api/products", func(w http.ResponseWriter, r *http.Request) {
		var req CreateProductRequest
		json.NewDecoder(r.Body).Decode(&req)

		resp, err := goodsClient.CreateProduct(context.Background(), &goodsPb.CreateProductRequest{
			Name: req.Name, Description: req.Description, Price: req.Price,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"id": resp.GetId()})
	})

	mux.HandleFunc("GET /api/products", func(w http.ResponseWriter, r *http.Request) {
		resp, err := goodsClient.ListProducts(context.Background(), &goodsPb.ListProductsRequest{})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp.GetProducts())
	})

	// --- Запуск ---
	log.Println("API Gateway is running on http://localhost:8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatalf("failed to serve HTTP: %v", err)
	}
}
