package service

import (
	"context"
	"errors"
	"time"

	"github.com/Sp1r14ual/ecommerce-go/auth-service/internal/repository"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	repo      *repository.AuthRepo
	jwtSecret string // Секретный ключ для подписи токена
}

func NewAuthService(repo *repository.AuthRepo, secret string) *AuthService {
	return &AuthService{repo: repo, jwtSecret: secret}
}

// Register хэширует пароль и передает данные в БД
func (s *AuthService) Register(ctx context.Context, email, password string) (int64, error) {
	// 1. Хэшируем пароль (10 - это "стоимость" хэширования, идеальный баланс секьюрности и скорости)
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 10)
	if err != nil {
		return 0, err
	}

	// 2. Сохраняем в базу
	return s.repo.CreateUser(ctx, email, string(hash))
}

// Login проверяет пароль и выдает JWT
func (s *AuthService) Login(ctx context.Context, email, password string) (string, error) {
	// 1. Достаем юзера из БД
	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		return "", err
	}

	// 2. Сравниваем введенный пароль с хэшем из БД
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		return "", errors.New("invalid credentials") // неверный пароль
	}

	// 3. Создаем JWT токен
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"uid":   user.ID,
		"email": user.Email,
		"exp":   time.Now().Add(time.Hour * 72).Unix(), // Токен живет 72 часа
	})

	// 4. Подписываем токен нашим секретным ключом
	tokenString, err := token.SignedString([]byte(s.jwtSecret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}
