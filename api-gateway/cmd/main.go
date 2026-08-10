package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"

	_ "github.com/Sp1r14ual/ecommerce-go/api-gateway/docs"

	authPb "github.com/Sp1r14ual/ecommerce-go/proto/auth"
	goodsPb "github.com/Sp1r14ual/ecommerce-go/proto/goods"
	orderPb "github.com/Sp1r14ual/ecommerce-go/proto/order"

	"github.com/Sp1r14ual/ecommerce-go/pkg/tracer"

	httpSwagger "github.com/swaggo/http-swagger/v2" // Swagger handler
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// @title E-Commerce API Gateway
// @version 1.0
// @description API Gateway for the Go microservices E-Commerce project.
// @host localhost:8080
// @BasePath /api

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Введите токен в формате: Bearer {твой_токен}

// --- СТРУКТУРЫ (Запросы и Ответы) ---
type AuthRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}
type CreateProductRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Price       int64  `json:"price"`
}
type CreateOrderRequest struct {
	ProductID string `json:"product_id"`
	Quantity  int32  `json:"quantity"`
}

// Структуры для ответов, чтобы Swagger красиво рисовал модели
type RegisterResponse struct {
	UserID int64 `json:"user_id"`
}
type LoginResponse struct {
	AccessToken string `json:"access_token"`
}
type ProductIDResponse struct {
	ID string `json:"id"`
}
type ProductResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Price       int64  `json:"price"`
}
type OrderResponse struct {
	OrderID int64  `json:"order_id"`
	Status  string `json:"status"`
}
type MessageResponse struct {
	Status string `json:"status"`
}
type ErrorResponse struct {
	Error string `json:"error"`
}

type contextKey string

const userIDKey contextKey = "user_id"

// --- СТРУКТУРА ХЕНДЛЕРА ---
type GatewayHandler struct {
	authClient  authPb.AuthServiceClient
	goodsClient goodsPb.GoodsServiceClient
	orderClient orderPb.OrderServiceClient
}

// @Summary Registration
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body AuthRequest true "Credentials"
// @Success 200 {object} RegisterResponse
// @Failure 500 {object} ErrorResponse
// @Router /register [post]
func (h *GatewayHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req AuthRequest
	json.NewDecoder(r.Body).Decode(&req)
	resp, err := h.authClient.Register(r.Context(), &authPb.RegisterRequest{Email: req.Email, Password: req.Password})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(RegisterResponse{UserID: resp.GetUserId()})
}

// @Summary Login
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body AuthRequest true "Credentials"
// @Success 200 {object} LoginResponse
// @Failure 401 {object} ErrorResponse
// @Router /login [post]
func (h *GatewayHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req AuthRequest
	json.NewDecoder(r.Body).Decode(&req)
	resp, err := h.authClient.Login(r.Context(), &authPb.LoginRequest{Email: req.Email, Password: req.Password})
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(LoginResponse{AccessToken: resp.GetAccessToken()})
}

// @Summary Logout
// @Tags Auth
// @Security BearerAuth
// @Produce json
// @Success 200 {object} MessageResponse
// @Router /logout [post]
func (h *GatewayHandler) Logout(w http.ResponseWriter, r *http.Request) {
	token := strings.Split(r.Header.Get("Authorization"), " ")[1]
	_, err := h.authClient.Logout(r.Context(), &authPb.LogoutRequest{AccessToken: token})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(MessageResponse{Status: "logged out and token revoked"})
}

// @Summary Get all products
// @Tags Products
// @Produce json
// @Success 200 {array} ProductResponse
// @Router /products [get]
func (h *GatewayHandler) ListProducts(w http.ResponseWriter, r *http.Request) {
	resp, err := h.goodsClient.ListProducts(r.Context(), &goodsPb.ListProductsRequest{})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp.GetProducts())
}

// @Summary Add a new product
// @Tags Products
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body CreateProductRequest true "Product INFO"
// @Success 200 {object} ProductIDResponse
// @Router /products [post]
func (h *GatewayHandler) CreateProduct(w http.ResponseWriter, r *http.Request) {
	var req CreateProductRequest
	json.NewDecoder(r.Body).Decode(&req)
	resp, err := h.goodsClient.CreateProduct(r.Context(), &goodsPb.CreateProductRequest{Name: req.Name, Description: req.Description, Price: req.Price})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ProductIDResponse{ID: resp.GetId()})
}

// @Summary Create an order
// @Tags Orders
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body CreateOrderRequest true "Order Details"
// @Success 200 {object} OrderResponse
// @Router /orders [post]
func (h *GatewayHandler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	var req CreateOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	userID := r.Context().Value(userIDKey).(int64)
	resp, err := h.orderClient.CreateOrder(r.Context(), &orderPb.CreateOrderRequest{
		UserId: userID, ProductId: req.ProductID, Quantity: req.Quantity})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(OrderResponse{OrderID: resp.GetOrderId(), Status: resp.GetStatus()})
}

// --- MIDDLEWARE ---
func authMiddleware(authClient authPb.AuthServiceClient, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
		resp, err := authClient.ValidateToken(r.Context(), &authPb.ValidateTokenRequest{AccessToken: parts[1]})
		if err != nil || !resp.GetIsValid() {
			http.Error(w, "unauthorized or token revoked", http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(r.Context(), userIDKey, resp.GetUserId())
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

// --- ИНИЦИАЛИЗАЦИЯ И МАРШРУТИЗАТОР ---
func main() {
	jaegerAddr := os.Getenv("JAEGER_ADDR")
	if jaegerAddr == "" {
		jaegerAddr = "localhost:4317"
	}
	tp, _ := tracer.InitTracer("api-gateway", jaegerAddr)
	defer tp.Shutdown(context.Background())

	authAddr := os.Getenv("AUTH_SERVICE_ADDR")
	if authAddr == "" {
		authAddr = "localhost:50051"
	}
	authConn, _ := grpc.NewClient(authAddr, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithStatsHandler(otelgrpc.NewClientHandler()))
	defer authConn.Close()

	goodsAddr := os.Getenv("GOODS_SERVICE_ADDR")
	if goodsAddr == "" {
		goodsAddr = "localhost:50052"
	}
	goodsConn, _ := grpc.NewClient(goodsAddr, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithStatsHandler(otelgrpc.NewClientHandler()))
	defer goodsConn.Close()

	orderAddr := os.Getenv("ORDER_SERVICE_ADDR")
	if orderAddr == "" {
		orderAddr = "localhost:50053"
	}
	orderConn, _ := grpc.NewClient(orderAddr, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithStatsHandler(otelgrpc.NewClientHandler()))
	defer orderConn.Close()

	// Инициализируем наш хендлер-обработчик
	h := &GatewayHandler{
		authClient:  authPb.NewAuthServiceClient(authConn),
		goodsClient: goodsPb.NewGoodsServiceClient(goodsConn),
		orderClient: orderPb.NewOrderServiceClient(orderConn),
	}

	mux := http.NewServeMux()

	// Ручки (с использованием методов хендлера)
	mux.HandleFunc("POST /api/register", h.Register)
	mux.HandleFunc("POST /api/login", h.Login)
	mux.HandleFunc("GET /api/products", h.ListProducts)

	// Защищенные ручки (заворачиваем методы хендлера в миддлварь)
	mux.HandleFunc("POST /api/logout", authMiddleware(h.authClient, h.Logout))
	mux.HandleFunc("POST /api/products", authMiddleware(h.authClient, h.CreateProduct))
	mux.HandleFunc("POST /api/orders", authMiddleware(h.authClient, h.CreateOrder))

	// Эндпоинты Swagger и Metrics
	mux.HandleFunc("GET /swagger/", httpSwagger.WrapHandler)

	log.Println("API Gateway is running on http://localhost:8080")
	log.Println("Swagger Docs available at http://localhost:8080/swagger/index.html")

	handler := otelhttp.NewHandler(mux, "api-gateway-http")
	if err := http.ListenAndServe(":8080", handler); err != nil {
		log.Fatalf("failed to serve HTTP: %v", err)
	}
}
