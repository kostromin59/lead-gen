package main

import (
	"context"
	"log"
	"log/slog"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	messagesemantic "github.com/kostromin59/lead-gen/internal/infrastructure/agents/message-semantic"
	"github.com/kostromin59/lead-gen/internal/repositories/messages"
)

func main() {
	if err := godotenv.Load(".env"); err != nil {
		slog.Warn(".env not found", slog.String("err", err.Error()))
	}

	pool, err := pgxpool.New(context.Background(), os.Getenv("DB_CONN"))
	if err != nil {
		panic(err)
	}

	messagesRepo := messages.New(pool)
	agent := messagesemantic.New()

	for {
		msgs, err := messagesRepo.FindNotHandledByAI(context.TODO())
		if err != nil {
			panic(err)
		}

		if len(msgs) == 0 {
			log.Printf("not found")
			return
		}

		descr := ""
		if msgs[0].ChatDescription != nil {
			descr = *msgs[0].ChatDescription
		}

		log.Println("start handle", msgs[0].ChatName, len(msgs))
		resp, err := agent.Handle(context.TODO(), msgs[0].ChatName, descr, msgs)
		if err != nil {
			panic(err)
		}

		for id, s := range resp {
			tid := "0"
			if msgs[0].ThreadID != nil {
				tid = *msgs[0].ThreadID
			}
			messagesRepo.UpdateSemantic(context.TODO(), id, msgs[0].ChatID, tid, s.Domain, s.Entities, s.Intent)
		}

		_ = resp
	}
}
