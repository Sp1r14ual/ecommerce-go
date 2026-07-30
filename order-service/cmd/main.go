package main

import (
	"context"
	"encoding/json"
	"fmt"
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

	"github.com/segmentio/kafka-go"
)

type OrderServer struct {
	pb.UnimplementedOrderServiceServer
	db          *pgxpool.Pool
	goodsClient goodsPb.GoodsServiceClient // Клиент для похода в Каталог
	kafkaWriter *kafka.Writer
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

	// Формируем сообщение (для простоты - JSON)
	eventMsg := map[string]interface{}{
		"order_id": orderID,
		"user_id":  req.GetUserId(),
		"status":   "CREATED",
	}
	eventBytes, _ := json.Marshal(eventMsg)

	// Пишем в топик "orders"
	err = s.kafkaWriter.WriteMessages(ctx, kafka.Message{
		Key:   []byte(fmt.Sprint(orderID)), // Ключ нужен, чтобы сообщения одного заказа шли по порядку
		Value: eventBytes,
	})
	if err != nil {
		// Даже если Kafka упала, заказ мы уже создали в БД.
		// Просто логируем ошибку, но не возвращаем её юзеру!
		log.Printf("Failed to write to Kafka: %v", err)
	} else {
		log.Printf("Order #%d event published to Kafka!", orderID)
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

	// --- 1.5 Подключение к Kafka ---
	kafkaAddr := os.Getenv("KAFKA_ADDR")
	if kafkaAddr == "" {
		kafkaAddr = "localhost:9092"
	}

	kw := &kafka.Writer{
		Addr:     kafka.TCP(kafkaAddr),
		Topic:    "orders", // Название нашего "канала"
		Balancer: &kafka.LeastBytes{},
	}
	defer kw.Close()

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
		kafkaWriter: kw,
	})

	reflection.Register(s) // Для тестов из Postman

	log.Println("Order gRPC Service is running on port 50053...")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
