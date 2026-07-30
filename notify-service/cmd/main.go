package main

import (
	"context"
	"encoding/json"
	"log"
	"os"

	"github.com/segmentio/kafka-go"
)

type OrderEvent struct {
	OrderID int64  `json:"order_id"`
	UserID  int64  `json:"user_id"`
	Status  string `json:"status"`
}

func main() {
	kafkaAddr := os.Getenv("KAFKA_ADDR")
	if kafkaAddr == "" {
		kafkaAddr = "localhost:9092"
	}

	// Создаем читателя (Consumer)
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{kafkaAddr},
		Topic:   "orders",
		GroupID: "notify-group", // Имя группы консьюмеров (важно для правильного масштабирования)
		// StartOffset: kafka.FirstOffset,

	})
	defer reader.Close()

	log.Println("Notify Service started. Waiting for Kafka messages...")

	for {
		// Блокируется и ждет новых сообщений
		msg, err := reader.ReadMessage(context.Background())
		if err != nil {
			log.Printf("Error reading message: %v", err)
			continue
		}

		var event OrderEvent
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			log.Printf("Error parsing message: %v", err)
			continue
		}

		// Имитируем бизнес-логику отправки Email / SMS
		log.Printf("🔔 [NOTIFY] Sending Email to User ID: %d! Event: Order #%d is %s\n",
			event.UserID, event.OrderID, event.Status)
	}
}
