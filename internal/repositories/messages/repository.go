package messages

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kostromin59/lead-gen/internal/models"
)

type repository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *repository {
	return &repository{
		pool: pool,
	}
}

func (r *repository) Create(ctx context.Context, dto models.CreateMessage) error {
	const op = "repositories.messages.Create"

	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel:   pgx.ReadCommitted,
		AccessMode: pgx.ReadWrite,
	})
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx,
		`INSERT INTO messages (
			chat_id,
			message_id, 
			thread_id,
			content,
			message_time,
			chat_name,
			chat_description,
			account_id
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8
		)`,
		dto.ChatID,
		dto.MessageID,
		dto.ThreadID,
		dto.Content,
		dto.MessageTime,
		dto.ChatName,
		dto.ChatDescription,
		dto.AccountID,
	); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}
