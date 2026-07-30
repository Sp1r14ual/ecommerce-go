package main

import (
	"context"
	"log"
	"net"
	"os"

	goodsPb "github.com/Sp1r14ual/ecommerce-go/proto/goods"
	pb "github.com/Sp1r14ual/ecommerce-go/proto/order"

	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

type OrderServer struct {
	pb.UnimplementedOrderServiceServer
	db          *pgxpool.Pool
	goodsClient goodsPb.GoodsServiceClient // Клиент для похода в Каталог
}

func (s *OrderServer) CreateOrder(ctx context.Context, req *pb.CreateOrderRequest) (*pb.CreateOrderResponse, error) {
	// 1. МЕЖСЕРВИСНЫЙ ВЫЗОВ: Идем в Каталог за товаром по gRPC
	product, err := s.goodsClient.GetProduct(ctx, &goodsPb.GetProductRequest{
		Id: req.GetProductId(),
	})
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "failed to get product from goods-service: %v", err)
	}

	// 2. БИЗНЕС-ЛОГИКА: Считаем итоговую цену (Цена * Количество)
	totalPrice := product.GetPrice() * int64(req.GetQuantity())

	// 3. СОХРАНЕНИЕ В БД:
	var orderID int64
	query := `INSERT INTO orders (user_id, product_id, quantity, total_price, status) VALUES ($1, $2, $3, $4, $5) RETURNING id`

	err = s.db.QueryRow(ctx, query, req.GetUserId(), req.GetProductId(), req.GetQuantity(), totalPrice, "CREATED").Scan(&orderID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create order in db: %v", err)
	}

	log.Printf("Order #%d created successfully for user %d!", orderID, req.GetUserId())

	return &pb.CreateOrderResponse{
		OrderId: orderID,
		Status:  "CREATED",
	}, nil
}

func main() {
	ctx := context.Background()

	// --- 1. Подключение к Базе Данных (Postgres) ---
	// В реальном проде для заказов делают отдельную базу,
	// но для разработки мы переиспользуем контейнер с auth_db
	connString := os.Getenv("DATABASE_URL")
	if connString == "" {
		connString = "postgres://auth_user:auth_password@localhost:5432/auth_db?sslmode=disable"
	}
	dbPool, err := pgxpool.New(ctx, connString)
	if err != nil {
		log.Fatalf("Unable to connect to DB: %v", err)
	}
	defer dbPool.Close()

	// Создаем таблицу
	_, err = dbPool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS orders (
			id SERIAL PRIMARY KEY,
			user_id INT NOT NULL,
			product_id VARCHAR(50) NOT NULL,
			quantity INT NOT NULL,
			total_price BIGINT NOT NULL,
			status VARCHAR(50) NOT NULL
		);
	`)
	if err != nil {
		log.Fatalf("Failed to create orders table: %v", err)
	}

	// --- 2. Подключение к Goods Service (Клиент) ---
	goodsAddr := os.Getenv("GOODS_SERVICE_ADDR")
	if goodsAddr == "" {
		goodsAddr = "localhost:50052"
	}
	goodsConn, err := grpc.NewClient(goodsAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to internal goods service: %v", err)
	}
	defer goodsConn.Close()
	goodsClient := goodsPb.NewGoodsServiceClient(goodsConn)

	// --- 3. Поднимаем наш собственный gRPC Сервер на ПОРТУ 50053 ---
	lis, err := net.Listen("tcp", ":50053")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterOrderServiceServer(s, &OrderServer{
		db:          dbPool,
		goodsClient: goodsClient,
	})

	reflection.Register(s) // Для тестов из Postman

	log.Println("Order gRPC Service is running on port 50053...")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
