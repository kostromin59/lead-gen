package qdrant

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/kostromin59/lead-gen/internal/models"
	"github.com/qdrant/go-client/qdrant"
	qd "github.com/qdrant/go-client/qdrant"
)

type repository struct {
	client         *qd.Client
	collectionName string
}

func New(host string, port int, collectionName string) (*repository, error) {
	// conn, err := grpc.NewClient(
	// 	fmt.Sprintf("%s:%d", host, port),
	// 	grpc.WithTransportCredentials(insecure.NewCredentials()),
	// )

	// if err != nil {
	// 	return nil, fmt.Errorf("failed to connect to Qdrant: %w", err)
	// }

	// client := qd.NewGrpcClientFromConn(conn)

	client, err := qd.NewClient(&qd.Config{
		Host: host,
		Port: port,
	})
	if err != nil {
		return nil, err
	}

	return &repository{
		client:         client,
		collectionName: collectionName,
	}, nil
}

// EnsureCollectionExists создает коллекцию, если она не существует
func (r *repository) EnsureCollectionExists(ctx context.Context, vectorSize uint64) error {
	// Проверяем существование коллекции
	collections, err := r.client.ListCollections(ctx)
	if err != nil {
		return fmt.Errorf("failed to list collections: %w", err)
	}

	for _, col := range collections {
		if col == r.collectionName {
			return nil // Коллекция уже существует
		}
	}

	// Создаем коллекцию
	err = r.client.CreateCollection(ctx, &qdrant.CreateCollection{
		CollectionName: r.collectionName,
		VectorsConfig: &qdrant.VectorsConfig{
			Config: &qdrant.VectorsConfig_ParamsMap{
				ParamsMap: &qdrant.VectorParamsMap{
					Map: map[string]*qdrant.VectorParams{
						"content": {
							Size:     vectorSize,
							Distance: qdrant.Distance_Cosine,
						},
						"domain": {
							Size:     vectorSize,
							Distance: qdrant.Distance_Cosine,
						},
						"entities": {
							Size:     vectorSize,
							Distance: qdrant.Distance_Cosine,
						},
						"intent": {
							Size:     vectorSize,
							Distance: qdrant.Distance_Cosine,
						},
					},
				},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create collection: %w", err)
	}

	return nil
}

func (r *repository) InsertMessages(ctx context.Context, messages []models.Message) error {
	const op = "repositories.qdrant.InsertMessages"

	if len(messages) == 0 {
		return nil
	}

	points := make([]*qdrant.PointStruct, 0, len(messages))
	mu := new(sync.Mutex)
	wg := new(sync.WaitGroup)
	for _, msg := range messages {
		wg.Go(func() {
			start := time.Now()
			contentVec, err := getEmbeddingFromOllama(fmt.Sprintf("Message from %s: %q", msg.SenderName, msg.Content))
			if err != nil {
				log.Printf("%+v", fmt.Errorf("%s: failed to embed content: %w", op, err))
				return
			}

			domainVec, err := getEmbeddingFromOllama(fmt.Sprintf("domain of the message: %s", strings.Join(msg.Domain, ",")))
			if err != nil {
				log.Printf("%+v", fmt.Errorf("%s: failed to embed domain: %w", op, err))
				return
			}

			entitiesVec, err := getEmbeddingFromOllama(fmt.Sprintf("entities of the message: %s", strings.Join(msg.Entities, ",")))
			if err != nil {
				log.Printf("%+v", fmt.Errorf("%s: failed to embed entities: %w", op, err))
				return
			}

			intentVec, err := getEmbeddingFromOllama(fmt.Sprintf("intent of the message: %s", strings.Join(msg.Intent, ",")))
			if err != nil {
				log.Printf("%+v", fmt.Errorf("%s: failed to embed intent: %w", op, err))
				return
			}

			threadID := "null"
			if msg.ThreadID != nil {
				threadID = *msg.ThreadID
			}

			pointID := &qdrant.PointId{
				PointIdOptions: &qdrant.PointId_Uuid{
					Uuid: uuid.NewString(),
				},
			}

			payload := map[string]*qdrant.Value{
				"chat_id": {
					Kind: &qdrant.Value_StringValue{
						StringValue: msg.ChatID,
					},
				},
				"message_id": {
					Kind: &qdrant.Value_StringValue{
						StringValue: msg.MessageID,
					},
				},
				"thread_id": {
					Kind: &qdrant.Value_StringValue{
						StringValue: threadID,
					},
				},
				"content": {
					Kind: &qdrant.Value_StringValue{
						StringValue: msg.Content,
					},
				},
				"message_time": {
					Kind: &qdrant.Value_StringValue{
						StringValue: msg.MessageTime.Format(time.RFC3339),
					},
				},
				"chat_name": {
					Kind: &qdrant.Value_StringValue{
						StringValue: msg.ChatName,
					},
				},
				"sender_id": {
					Kind: &qdrant.Value_StringValue{
						StringValue: msg.SenderID,
					},
				},
				"sender_name": {
					Kind: &qdrant.Value_StringValue{
						StringValue: msg.SenderName,
					},
				},
				"account_id": {
					Kind: &qdrant.Value_StringValue{
						StringValue: msg.AccountID,
					},
				},
				"created_at": {
					Kind: &qdrant.Value_StringValue{
						StringValue: msg.CreatedAt.Format(time.RFC3339),
					},
				},
				"is_ai_handled": {
					Kind: &qdrant.Value_BoolValue{
						BoolValue: msg.IsAIHandled,
					},
				},
			}

			if msg.ChatDescription != nil {
				payload["chat_description"] = &qdrant.Value{
					Kind: &qdrant.Value_StringValue{
						StringValue: *msg.ChatDescription,
					},
				}
			}

			if len(msg.Domain) > 0 {
				domainValues := make([]*qdrant.Value, len(msg.Domain))
				for i, val := range msg.Domain {
					domainValues[i] = &qdrant.Value{
						Kind: &qdrant.Value_StringValue{
							StringValue: val,
						},
					}
				}
				payload["domain"] = &qdrant.Value{
					Kind: &qdrant.Value_ListValue{
						ListValue: &qdrant.ListValue{
							Values: domainValues,
						},
					},
				}
			}

			if len(msg.Entities) > 0 {
				entitiesValues := make([]*qdrant.Value, len(msg.Entities))
				for i, val := range msg.Entities {
					entitiesValues[i] = &qdrant.Value{
						Kind: &qdrant.Value_StringValue{
							StringValue: val,
						},
					}
				}
				payload["entities"] = &qdrant.Value{
					Kind: &qdrant.Value_ListValue{
						ListValue: &qdrant.ListValue{
							Values: entitiesValues,
						},
					},
				}
			}

			if len(msg.Intent) > 0 {
				intentValues := make([]*qdrant.Value, len(msg.Intent))
				for i, val := range msg.Intent {
					intentValues[i] = &qdrant.Value{
						Kind: &qdrant.Value_StringValue{
							StringValue: val,
						},
					}
				}
				payload["intent"] = &qdrant.Value{
					Kind: &qdrant.Value_ListValue{
						ListValue: &qdrant.ListValue{
							Values: intentValues,
						},
					},
				}
			}

			for i, s := range msg.Domain {
				msg.Domain[i] = strings.ToLower(s)
			}

			for i, s := range msg.Entities {
				msg.Entities[i] = strings.ToLower(s)
			}

			for i, s := range msg.Intent {
				msg.Intent[i] = strings.ToLower(s)
			}

			point := &qdrant.PointStruct{
				Id: pointID,
				Vectors: qdrant.NewVectorsMap(map[string]*qd.Vector{
					"content":  qdrant.NewVector(contentVec...),
					"domain":   qdrant.NewVector(domainVec...),
					"entities": qdrant.NewVector(entitiesVec...),
					"intent":   qdrant.NewVector(intentVec...),
				}),
				Payload: payload,
			}

			log.Println("point", time.Since(start))

			mu.Lock()
			points = append(points, point)
			mu.Unlock()
		})
	}
	wg.Wait()

	upsertReq := &qdrant.UpsertPoints{
		CollectionName: r.collectionName,
		Points:         points,
	}

	if _, err := r.client.Upsert(ctx, upsertReq); err != nil {
		return fmt.Errorf("%s: failed to upsert points: %w", op, err)
	}

	return nil
}

func (r *repository) InsertMessagesBatch(ctx context.Context, messages []models.Message, batchSize int) error {
	const op = "repositories.qdrant.InsertMessagesBatch"

	if batchSize <= 0 {
		batchSize = 100
	}

	for i := 0; i < len(messages); i += batchSize {
		end := i + batchSize
		if end > len(messages) {
			end = len(messages)
		}

		batch := messages[i:end]
		start := time.Now()
		if err := r.InsertMessages(ctx, batch); err != nil {
			return fmt.Errorf("%s: failed to insert batch %d-%d: %w", op, i, end, err)
		}
		log.Println("==========================")
		log.Println("batch", i, time.Since(start))

	}

	return nil
}

func (r *repository) Close() error {
	if r.client != nil {
		return r.client.Close()
	}
	return nil
}

// getEmbeddingFromOllama получает эмбеддинг из Ollama
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
