// microservices/events/main.go
package main

import (
    "context"
    "encoding/json"
    "log"
    "net/http"
    "os"
    "time"

    "github.com/segmentio/kafka-go"
)

type Event struct {
    Type    string      `json:"type"`
    Payload interface{} `json:"payload"`
}

var writer *kafka.Writer

func main() {
    broker := getEnv("KAFKA_BROKERS", "kafka:9092")
    writer = &kafka.Writer{
        Addr:     kafka.TCP(broker),
        Balancer: &kafka.LeastBytes{},
    }

    // Запускаем consumer
    go consume()

    // Health check
    http.HandleFunc("/api/events/health", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(map[string]bool{"status": true})
    })

    // Обработчики событий
    http.HandleFunc("/api/events/movie", makeEventHandler("MovieCreated"))
    http.HandleFunc("/api/events/user", makeEventHandler("UserCreated"))
    http.HandleFunc("/api/events/payment", makeEventHandler("PaymentProcessed"))

    port := getEnv("PORT", "8082")
    log.Printf("✅ Events service запущен на :%s", port)
    log.Fatal(http.ListenAndServe(":"+port, nil))
}

func getEnv(key, fallback string) string {
    if v, ok := os.LookupEnv(key); ok {
        return v
    }
    return fallback
}

func makeEventHandler(topic string) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        var payload map[string]interface{}
        if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
            http.Error(w, err.Error(), http.StatusBadRequest)
            return
        }

        event := Event{
            Type:    topic,
            Payload: payload,
        }

        value, _ := json.Marshal(event)

        err := writer.WriteMessages(context.Background(), kafka.Message{
            Topic: topic,
            Value: value,
        })

        if err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }

        w.WriteHeader(http.StatusAccepted)
    }
}

func consume() {
    broker := getEnv("KAFKA_BROKERS", "kafka:9092")
    for _, topic := range []string{"MovieCreated", "UserCreated", "PaymentProcessed"} {
        go func(topic string) {
            for {
                reader := kafka.NewReader(kafka.ReaderConfig{
                    Brokers: []string{broker},
                    Topic:   topic,
                    GroupID: "events-group",
                })

                msg, err := reader.ReadMessage(context.Background())
                if err != nil {
                    log.Printf("❌ Ошибка при чтении из %s: %v", topic, err)
                    time.Sleep(5 * time.Second) // Подождать перед повторной попыткой
                    continue
                }
                log.Printf("[Kafka Consumer] Topic: %s | Value: %s", msg.Topic, string(msg.Value))
                reader.Close()
            }
        }(topic)
    }
}