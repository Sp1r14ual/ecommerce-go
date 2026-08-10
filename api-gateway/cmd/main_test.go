package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	authPb "github.com/Sp1r14ual/ecommerce-go/proto/auth"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
)

// 1. СОЗДАЕМ МОК (MOCK)
// Нам не нужно поднимать реальный Auth Service, базу Postgres и gRPC сервер для теста!
// Мы создаем фальшивый (поддельный) клиент, который просто делает вид, что он gRPC.
type MockAuthClient struct {
	mock.Mock
	authPb.AuthServiceClient // Встраиваем интерфейс, чтобы удовлетворить компилятор
}

// Переопределяем метод Register нашей фальшивки
func (m *MockAuthClient) Register(ctx context.Context, in *authPb.RegisterRequest, opts ...grpc.CallOption) (*authPb.RegisterResponse, error) {
	args := m.Called(ctx, in)
	return args.Get(0).(*authPb.RegisterResponse), args.Error(1)
}

// 2. ПИШЕМ САМ ТЕСТ
func TestGatewayHandler_Register_Success(t *testing.T) {
	// Инициализируем нашу подделку
	mockAuth := new(MockAuthClient)

	// Программируем мок:
	// "Если функция Register вызвана с email=test@test.com и password=123,
	// то ВЕРНИ УСПЕШНО сгенерированный UserId = 42 без ошибок (nil)".
	mockAuth.On("Register", mock.Anything, &authPb.RegisterRequest{
		Email:    "test@test.com",
		Password: "123",
	}).Return(&authPb.RegisterResponse{UserId: 42}, nil)

	// 3. ПОДГОТОВКА СРЕДЫ ДЛЯ ШЛЮЗА
	// Подсовываем шлюзу наш поддельный клиент вместо настоящего gRPC-соединения.
	h := &GatewayHandler{
		authClient: mockAuth,
	}

	// 4. СИМУЛИРУЕМ ЗАПРОС (Имитируем Postman)
	requestBody := []byte(`{"email":"test@test.com","password":"123"}`)

	// Создаем фальшивый HTTP запрос (req)
	req, _ := http.NewRequest("POST", "/api/register", bytes.NewBuffer(requestBody))

	// И создаем фальшивый объект ответа (rr - ResponseRecorder), куда функция запишет результат
	rr := httptest.NewRecorder()

	// 5. ВЫПОЛНЕНИЕ (ДЕРГАЕМ НАШ РЕАЛЬНЫЙ КОД)
	h.Register(rr, req)

	// 6. ПРОВЕРКИ (ASSERT O'CLOCK)

	// Проверяем, что HTTP код возврата - 200 OK
	assert.Equal(t, http.StatusOK, rr.Code)

	// Парсим JSON, который плюнул наш хендлер, и проверяем, что в нем ID = 42
	var response RegisterResponse
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	assert.NoError(t, err) // Убеждаемся, что JSON распарсился без ошибок

	// Самая главная проверка!
	assert.Equal(t, int64(42), response.UserID)

	// Проверяем, что фальшивый метод точно был вызван
	mockAuth.AssertExpectations(t)
}
