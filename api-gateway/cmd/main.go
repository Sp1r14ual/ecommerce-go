package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"

	authPb "github.com/Sp1r14ual/ecommerce-go/proto/auth"
	goodsPb "github.com/Sp1r14ual/ecommerce-go/proto/goods"
	orderPb "github.com/Sp1r14ual/ecommerce-go/proto/order"

	"github.com/Sp1r14ual/ecommerce-go/pkg/tracer"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// --- СТРУКТУРЫ ---
type AuthRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type CreateProductRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Price       int64  `json:"price"`
}

// Теперь клиент не передает user_id, мы берем его из токена!
type CreateOrderRequest struct { 
	ProductID string `json:"product_id"`
	Quantity  int32  `json:"quantity"`
}

// Кастомный тип для ключа контекста (хорошая практика в Go)
type contextKey string
const userIDKey contextKey = "user_id"

// --- MIDDLEWARE ДЛЯ ПРОВЕРКИ ТОКЕНА ---
func authMiddleware(authClient authPb.AuthServiceClient, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. Достаем заголовок "Authorization: Bearer <token>"
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "missing authorization header", http.StatusUnauthorized)
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, "invalid token format", http.StatusUnauthorized)
			return
		}
		token := parts[1]

		// 2. Стучимся в Auth Service по gRPC для проверки (включая проверку в Redis)
		resp, err := authClient.ValidateToken(r.Context(), &authPb.ValidateTokenRequest{
			AccessToken: token,
		})

		if err != nil || !resp.GetIsValid() {
			http.Error(w, "unauthorized or token revoked", http.StatusUnauthorized)
			return
		}

		// 3. Кладем user_id в контекст запроса и передаем дальше
		ctx := context.WithValue(r.Context(), userIDKey, resp.GetUserId())
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

func main() {
	// Инициализация трейсера Jaeger
	jaegerAddr := os.Getenv("JAEGER_ADDR")
	if jaegerAddr == "" { jaegerAddr = "localhost:4317" }
	tp, _ := tracer.InitTracer("api-gateway", jaegerAddr)
	defer tp.Shutdown(context.Background())

	// Подключения к gRPC сервисам (с трейсингом)
	authAddr := os.Getenv("AUTH_SERVICE_ADDR")
	if authAddr == "" { authAddr = "localhost:50051" }
	authConn, _ := grpc.NewClient(authAddr, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithStatsHandler(otelgrpc.NewClientHandler()))
	defer authConn.Close()
	authClient := authPb.NewAuthServiceClient(authConn)

	goodsAddr := os.Getenv("GOODS_SERVICE_ADDR")
	if goodsAddr == "" { goodsAddr = "localhost:50052" }
	goodsConn, _ := grpc.NewClient(goodsAddr, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithStatsHandler(otelgrpc.NewClientHandler()))
	defer goodsConn.Close()
	goodsClient := goodsPb.NewGoodsServiceClient(goodsConn)

	orderAddr := os.Getenv("ORDER_SERVICE_ADDR")
	if orderAddr == "" { orderAddr = "localhost:50053" }
	orderConn, _ := grpc.NewClient(orderAddr, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithStatsHandler(otelgrpc.NewClientHandler()))
	defer orderConn.Close()
	orderClient := orderPb.NewOrderServiceClient(orderConn)

	mux := http.NewServeMux()

	// --- ПУБЛИЧНЫЕ РУЧКИ (Без токена) ---
	mux.HandleFunc("POST /api/register", func(w http.ResponseWriter, r *http.Request) {
		var req AuthRequest
		json.NewDecoder(r.Body).Decode(&req)
		resp, err := authClient.Register(r.Context(), &authPb.RegisterRequest{Email: req.Email, Password: req.Password})
		if err != nil { http.Error(w, err.Error(), http.StatusInternalServerError); return }
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"user_id": resp.GetUserId()})
	})

	mux.HandleFunc("POST /api/login", func(w http.ResponseWriter, r *http.Request) {
		var req AuthRequest
		json.NewDecoder(r.Body).Decode(&req)
		resp, err := authClient.Login(r.Context(), &authPb.LoginRequest{Email: req.Email, Password: req.Password})
		if err != nil { http.Error(w, "Unauthorized", http.StatusUnauthorized); return }
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"access_token": resp.GetAccessToken()})
	})

	mux.HandleFunc("GET /api/products", func(w http.ResponseWriter, r *http.Request) {
		resp, err := goodsClient.ListProducts(r.Context(), &goodsPb.ListProductsRequest{})
		if err != nil { http.Error(w, err.Error(), http.StatusInternalServerError); return }
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp.GetProducts())
	})

	// --- ЗАЩИЩЕННЫЕ РУЧКИ (Нужен JWT) ---
	// Заворачиваем функцию-обработчик в нашу authMiddleware

	// 1. Выход (Logout)
	mux.HandleFunc("POST /api/logout", authMiddleware(authClient, func(w http.ResponseWriter, r *http.Request) {
		token := strings.Split(r.Header.Get("Authorization"), " ")[1]
		_, err := authClient.Logout(r.Context(), &authPb.LogoutRequest{AccessToken: token})
		if err != nil { http.Error(w, err.Error(), http.StatusInternalServerError); return }
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"status": "logged out and token revoked"})
	}))

	// 2. Создание товара (допустим, это могут делать только авторизованные продавцы)
	mux.HandleFunc("POST /api/products", authMiddleware(authClient, func(w http.ResponseWriter, r *http.Request) {
		var req CreateProductRequest
		json.NewDecoder(r.Body).Decode(&req)
		resp, err := goodsClient.CreateProduct(r.Context(), &goodsPb.CreateProductRequest{Name: req.Name, Description: req.Description, Price: req.Price})
		if err != nil { http.Error(w, err.Error(), http.StatusInternalServerError); return }
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"id": resp.GetId()})
	}))

	// 3. Создание заказа (user_id берем из контекста, а не из JSON)
	mux.HandleFunc("POST /api/orders", authMiddleware(authClient, func(w http.ResponseWriter, r *http.Request) {
		var req CreateOrderRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest); return
		}

		// ДОСТАЕМ ИЗ КОНТЕКСТА ID АВТОРИЗОВАННОГО ПОЛЬЗОВАТЕЛЯ
		userID := r.Context().Value(userIDKey).(int64)

		resp, err := orderClient.CreateOrder(r.Context(), &orderPb.CreateOrderRequest{
			UserId:    userID, // <-- Безопасно!
			ProductId: req.ProductID,
			Quantity:  req.Quantity,
		})
		if err != nil { http.Error(w, err.Error(), http.StatusInternalServerError); return }
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"order_id": resp.GetOrderId(), "status": resp.GetStatus()})
	}))

	log.Println("API Gateway is running on http://localhost:8080")
	handler := otelhttp.NewHandler(mux, "api-gateway-http")
	if err := http.ListenAndServe(":8080", handler); err != nil {
		log.Fatalf("failed to serve HTTP: %v", err)
	}
}