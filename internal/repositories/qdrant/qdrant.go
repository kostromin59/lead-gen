package qdrant

import (
	"context"
	"fmt"
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
			Config: &qdrant.VectorsConfig_Params{
				Params: &qdrant.VectorParams{
					Size:     vectorSize,
					Distance: qdrant.Distance_Cosine,
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

	for _, msg := range messages {
		threadID := "null"
		if msg.ThreadID != nil {
			threadID = *msg.ThreadID
		}

		pointID := &qdrant.PointId{
			PointIdOptions: &qdrant.PointId_Uuid{
				Uuid: uuid.NewString(),
			},
		}

		vectorData := make([]float32, 384) // Размер должен соответствовать коллекции
		vector := qdrant.NewVector(vectorData...)

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

		point := &qdrant.PointStruct{
			Id: pointID,
			Vectors: &qdrant.Vectors{
				VectorsOptions: &qdrant.Vectors_Vector{
					Vector: vector,
				},
			},
			Payload: payload,
		}

		points = append(points, point)
	}

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
		if err := r.InsertMessages(ctx, batch); err != nil {
			return fmt.Errorf("%s: failed to insert batch %d-%d: %w", op, i, end, err)
		}
	}

	return nil
}

func (r *repository) Close() error {
	if r.client != nil {
		return r.client.Close()
	}
	return nil
}
