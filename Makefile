# .PHONY означает, что это имена команд, а не названия файлов, 
# чтобы Make не путался, если у тебя вдруг появится папка с именем "logs"
.PHONY: up down clean logs proto tidy help migrate-auth migrate-order swagger test

# --- DOCKER КОМАНДЫ ---

# Поднять весь проект (с пересборкой)
up:
	docker compose up -d --build

# Остановить весь проект
down:
	docker compose down

# Остановить проект и удалить все базы данных (полный сброс)
clean:
	docker compose down -v

# Смотреть все логи проекта в реальном времени
logs:
	docker compose logs -f auth-service goods-service order-service notify-service payment-service delivery-service api-gateway

# --- GO КОМАНДЫ ---

# Причесать зависимости
tidy:
	go mod tidy

# --- PROTOBUF КОМАНДЫ ---

# Сгенерировать Go-код из всех .proto файлов
proto:
	protoc -I proto/auth proto/auth/auth.proto --go_out=./proto/auth --go_opt=paths=source_relative --go-grpc_out=./proto/auth --go-grpc_opt=paths=source_relative
	protoc -I proto/goods proto/goods/goods.proto --go_out=./proto/goods --go_opt=paths=source_relative --go-grpc_out=./proto/goods --go-grpc_opt=paths=source_relative
	protoc -I proto/order proto/order/order.proto --go_out=./proto/order --go_opt=paths=source_relative --go-grpc_out=./proto/order --go-grpc_opt=paths=source_relative
	protoc -I proto/payment proto/payment/payment.proto --go_out=./proto/payment --go_opt=paths=source_relative --go-grpc_out=./proto/payment --go-grpc_opt=paths=source_relative
	@echo "Protobuf files generated successfully!"

# --- MIGRATIONS КОМАНДЫ ---

# Создать новую миграцию для Auth Service. 
# Пример использования в терминале: make migrate-auth name=add_phone_number
migrate-auth:
	migrate create -ext sql -dir ./auth-service/migrations -seq $(name)

# Создать новую миграцию для Order Service.
migrate-order:
	migrate create -ext sql -dir ./order-service/migrations -seq $(name)

# --- SWAGGER КОМАНДЫ ---

# Сгенерировать документацию Swagger для API Gateway
swagger:
	swag init -g api-gateway/cmd/main.go -o api-gateway/docs
	@echo "Swagger docs generated successfully!"

test:
	go test -v ./api-gateway/cmd/...

# --- ПОМОЩЬ ---

# Вывести список всех доступных команд (срабатывает, если просто написать make)
help:
	@echo "Доступные команды:"
	@echo "  make up             - Собрать и запустить все контейнеры (в фоне)"
	@echo "  make down           - Остановить все контейнеры"
	@echo "  make clean          - Удалить контейнеры и сбросить БД (удалить volumes)"
	@echo "  make logs           - Смотреть логи всех сервисов в реальном времени"
	@echo "  make proto          - Сгенерировать Go файлы из Protobuf контрактов"
	@echo "  make tidy           - Загрузить и обновить Go зависимости"
	@echo "  make migrate-auth name=X  - Создать новый файл миграции для Auth Service"
	@echo "  make migrate-order name=X - Создать новый файл миграции для Order Service"