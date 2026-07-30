package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/Sp1r14ual/ecommerce-go/auth-service/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrUserNotFound = errors.New("user not found")
var ErrUserExists = errors.New("user already exists")

type AuthRepo struct {
	db *pgxpool.Pool
}

func NewAuthRepo(db *pgxpool.Pool) *AuthRepo {
	return &AuthRepo{db: db}
}

// CreateUser сохраняет нового юзера в БД и возвращает его ID
func (r *AuthRepo) CreateUser(ctx context.Context, email string, passwordHash string) (int64, error) {
	query := `INSERT INTO users (email, password_hash) VALUES ($1, $2) RETURNING id`

	var id int64
	err := r.db.QueryRow(ctx, query, email, passwordHash).Scan(&id)
	if err != nil {
		// Очень базовая проверка на уникальность email (по хорошему надо парсить код ошибки Postgres)
		return 0, fmt.Errorf("%w: %v", ErrUserExists, err)
	}

	return id, nil
}

// GetUserByEmail ищет пользователя для проверки логина
func (r *AuthRepo) GetUserByEmail(ctx context.Context, email string) (domain.User, error) {
	query := `SELECT id, email, password_hash FROM users WHERE email = $1`

	var user domain.User
	err := r.db.QueryRow(ctx, query, email).Scan(&user.ID, &user.Email, &user.PasswordHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.User{}, ErrUserNotFound
		}
		return domain.User{}, err
	}

	return user, nil
}
