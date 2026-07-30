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

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{kafkaAddr},
		Topic:   "orders",
		// ВАЖНО: имя группы должно отличаться от notify-service!
		GroupID: "delivery-group",
		// StartOffset: kafka.FirstOffset,
	})
	defer reader.Close()

	log.Println("Delivery Service started. Waiting for paid orders...")

	for {
		msg, err := reader.ReadMessage(context.Background())
		if err != nil {
			log.Printf("Error reading message: %v", err)
			continue
		}

		var event OrderEvent
		json.Unmarshal(msg.Value, &event)

		if event.Status == "PAID" {
			log.Printf("📦 [DELIVERY] Starting packing process for Order #%d! Courier is assigned.\n", event.OrderID)
		}
	}
}
