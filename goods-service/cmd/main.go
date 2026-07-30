package main

import (
	"context"
	"log"
	"net"

	"github.com/Sp1r14ual/ecommerce-go/goods-service/internal/domain"
	"github.com/Sp1r14ual/ecommerce-go/goods-service/internal/repository"
	pb "github.com/Sp1r14ual/ecommerce-go/proto/goods"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// GoodsServer реализует наш gRPC контракт
type GoodsServer struct {
	pb.UnimplementedGoodsServiceServer
	repo *repository.GoodsRepo
}

func (s *GoodsServer) CreateProduct(ctx context.Context, req *pb.CreateProductRequest) (*pb.CreateProductResponse, error) {
	p := &domain.Product{
		Name:        req.GetName(),
		Description: req.GetDescription(),
		Price:       req.GetPrice(),
	}

	id, err := s.repo.Create(ctx, p)
	if err != nil {
		return nil, err
	}

	return &pb.CreateProductResponse{Id: id}, nil
}

func (s *GoodsServer) ListProducts(ctx context.Context, req *pb.ListProductsRequest) (*pb.ListProductsResponse, error) {
	products, err := s.repo.ListAll(ctx)
	if err != nil {
		return nil, err
	}

	// Перекладываем из доменной модели в модель protobuf
	var pbProducts []*pb.Product
	for _, p := range products {
		pbProducts = append(pbProducts, &pb.Product{
			Id:          p.ID.Hex(),
			Name:        p.Name,
			Description: p.Description,
			Price:       p.Price,
		})
	}

	return &pb.ListProductsResponse{Products: pbProducts}, nil
}

func main() {
	// Подключаемся к MongoDB (в докере)
	ctx := context.Background()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer client.Disconnect(ctx)

	// Выбираем БД "ecommerce"
	db := client.Database("ecommerce")
	repo := repository.NewGoodsRepo(db)

	lis, err := net.Listen("tcp", ":50052") // ПОРТ 50052 !!
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterGoodsServiceServer(s, &GoodsServer{repo: repo})

	// Включаем рефлексию, чтобы можно было тестировать из Postman
	reflection.Register(s)

	log.Println("Goods gRPC Service is running on port 50052...")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
