package main

import (
	"context"
	"log"
	"net"

	pb "github.com/Sp1r14ual/ecommerce-go/proto/payment"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type PaymentServer struct {
	pb.UnimplementedPaymentServiceServer
}

func (s *PaymentServer) ProcessPayment(ctx context.Context, req *pb.PaymentRequest) (*pb.PaymentResponse, error) {
	// Здесь могла бы быть интеграция со Stripe, ЮKassa или другим эквайрингом
	log.Printf("💳 Processing payment of %d cents for User %d...", req.GetAmount(), req.GetUserId())

	// Имитируем успешную оплату
	return &pb.PaymentResponse{
		Success:       true,
		TransactionId: "txn_87654321_mock",
	}, nil
}

func main() {
	lis, err := net.Listen("tcp", ":50054") // ПОРТ 50054 !!!
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterPaymentServiceServer(s, &PaymentServer{})
	reflection.Register(s)

	log.Println("Payment gRPC Service is running on port 50054...")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
