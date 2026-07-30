package server

import (
	"context"

	"github.com/Sp1r14ual/ecommerce-go/auth-service/internal/service"
	pb "github.com/Sp1r14ual/ecommerce-go/proto/auth"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// AuthServer реализует интерфейс, который сгенерировал protoc
type AuthServer struct {
	pb.UnimplementedAuthServiceServer
	authService *service.AuthService
}

func NewAuthServer(authService *service.AuthService) *AuthServer {
	return &AuthServer{authService: authService}
}

// Метод Register (строго совпадает с protobuf)
func (s *AuthServer) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	// Валидация (минимальная для примера)
	if req.GetEmail() == "" || req.GetPassword() == "" {
		return nil, status.Error(codes.InvalidArgument, "email and password are required")
	}

	userID, err := s.authService.Register(ctx, req.GetEmail(), req.GetPassword())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to register user: %v", err)
	}

	return &pb.RegisterResponse{UserId: userID}, nil
}

// Метод Login
func (s *AuthServer) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	token, err := s.authService.Login(ctx, req.GetEmail(), req.GetPassword())
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "failed to login: %v", err)
	}

	return &pb.LoginResponse{AccessToken: token}, nil
}
