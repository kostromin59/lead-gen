package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/joho/godotenv"
	"github.com/qdrant/go-client/qdrant"
)

func main() {
	if err := godotenv.Load(".env"); err != nil {
		slog.Warn(".env not found", slog.String("err", err.Error()))
	}

	client, err := qdrant.NewClient(&qdrant.Config{
		Host: "qdrant",
		Port: 6334,
	})
	if err != nil {
		panic(err)
	}

	vec, err := getEmbeddingFromOllama("авто")
	if err != nil {
		panic(err)
	}

	result, err := client.Query(context.Background(), &qdrant.QueryPoints{
		CollectionName: "messages",
		Using:          new("entities"),
		Query:          qdrant.NewQueryDense(vec), // искомый вектор
		WithPayload:    qdrant.NewWithPayloadEnable(true),
		WithVectors:    qdrant.NewWithVectors(false),
	})
	if err != nil {
		panic(err)
	}

	for i, point := range result {
		// Извлекаем ID
		id := point.Id.GetUuid()

		// Извлекаем payload
		payload := point.Payload

		// Получаем поля с безопасной проверкой
		content := getString(payload, "content")
		senderName := getString(payload, "sender_name")
		chatName := getString(payload, "chat_name")
		domain := getStringSlice(payload, "domain")
		intent := getStringSlice(payload, "intent")
		entities := getStringSlice(payload, "entities")
		messageTime := getString(payload, "message_time")

		// Форматируем вывод
		fmt.Printf("\n╔═══════════════════════════════════════════════════════════╗\n")
		fmt.Printf("║  🎯 Результат #%d  (Score: %.4f)                        ║\n", i+1, point.Score)
		fmt.Printf("╠═══════════════════════════════════════════════════════════╣\n")
		fmt.Printf("║  📝 ID: %s\n", id)
		fmt.Printf("║  💬 Сообщение: %s\n", content)
		fmt.Printf("║  👤 Отправитель: %s\n", senderName)
		fmt.Printf("║  📱 Чат: %s\n", chatName)
		fmt.Printf("║  🕐 Время: %s\n", messageTime)
		fmt.Printf("║  🏷️  Домен: %v\n", domain)
		fmt.Printf("║  🎯 Интент: %v\n", intent)
		fmt.Printf("║  👥 Сущности: %v\n", entities)
		fmt.Printf("╚═══════════════════════════════════════════════════════════╝\n")
	}
}

func getEmbeddingFromOllama(text string) ([]float32, error) {
	// Структура запроса
	type embedRequest struct {
		Model string `json:"model"`
		Input string `json:"input"`
	}

	// Структура ответа
	type embedResponse struct {
		Embeddings [][]float32 `json:"embeddings"`
	}

	// Формируем запрос
	reqBody := embedRequest{
		Model: "all-minilm:l6-v2",
		Input: text,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	// Отправляем запрос
	resp, err := http.Post("http://192.168.1.100:11434/api/embed", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Проверяем статус
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Ollama error: %s", string(body))
	}

	// Парсим ответ
	var result embedResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if len(result.Embeddings) == 0 || len(result.Embeddings[0]) == 0 {
		return nil, fmt.Errorf("empty embedding returned")
	}

	return result.Embeddings[0], nil
}

// Безопасное извлечение строки из payload
func getString(payload map[string]*qdrant.Value, key string) string {
	if payload == nil {
		return "-"
	}
	if val, ok := payload[key]; ok && val != nil {
		return val.GetStringValue()
	}
	return "-"
}

// Безопасное извлечение слайса строк из payload
func getStringSlice(payload map[string]*qdrant.Value, key string) []string {
	if payload == nil {
		return []string{}
	}
	if val, ok := payload[key]; ok && val != nil {
		listVal := val.GetListValue()
		if listVal == nil {
			return []string{}
		}
		result := make([]string, 0, len(listVal.Values))
		for _, v := range listVal.Values {
			result = append(result, v.GetStringValue())
		}
		return result
	}
	return []string{}
}
