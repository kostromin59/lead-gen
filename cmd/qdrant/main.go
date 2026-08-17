package main

import (
	"context"
	"log"
	"log/slog"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/kostromin59/lead-gen/internal/repositories/messages"
	"github.com/kostromin59/lead-gen/internal/repositories/qdrant"
)

func main() {
	if err := godotenv.Load(".env"); err != nil {
		slog.Warn(".env not found", slog.String("err", err.Error()))
	}

	pool, err := pgxpool.New(context.Background(), os.Getenv("DB_CONN"))
	if err != nil {
		panic(err)
	}

	repo, err := qdrant.New("qdrant", 6334, "messages")
	if err != nil {
		log.Fatal(err)
	}
	defer repo.Close()

	if err := repo.EnsureCollectionExists(context.Background(), 384); err != nil {
		panic(err)
	}

	messagesRepo := messages.New(pool)
	messages, err := messagesRepo.FindHandledByAI(context.Background())
	if err != nil {
		panic(err)
	}

	if err := repo.InsertMessagesBatch(context.Background(), messages, 100); err != nil {
		panic(err)
	}
}
