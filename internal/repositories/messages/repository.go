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
			sender_id,
			sender_name,
			account_id
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10
		)`,
		dto.ChatID,
		dto.MessageID,
		dto.ThreadID,
		dto.Content,
		dto.MessageTime,
		dto.ChatName,
		dto.ChatDescription,
		dto.SenderID,
		dto.SenderName,
		dto.AccountID,
	); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (r *repository) FindNotHandledByAI(ctx context.Context) ([]models.Message, error) {
	const op = "repositories.messages.FindNotHandledByAI"

	rows, err := r.pool.Query(ctx, `
		WITH random_chat_thread AS (
    -- Выбираем случайную комбинацию (chat_id, thread_id)
    SELECT 
        chat_id,
        thread_id
    FROM messages
    WHERE is_ai_handled = false 
        AND content != ''
    GROUP BY chat_id, thread_id
    ORDER BY RANDOM()
    LIMIT 1
)
SELECT 
    m.chat_id,
    m.message_id,
    m.thread_id,
    m.content,
    m.message_time,
    m.chat_name,
    m.chat_description,
    m.sender_id,
    m.sender_name,
    m.account_id,
    m.created_at,
    m.domain,
    m.entities,
    m.intent,
    m.is_ai_handled
FROM messages m
INNER JOIN random_chat_thread rct 
    ON m.chat_id = rct.chat_id 
    AND m.thread_id = rct.thread_id
WHERE m.is_ai_handled = false 
    AND m.content != ''
ORDER BY m.message_time ASC  -- сортировка по времени
LIMIT 50
	`)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var messages []models.Message
	for rows.Next() {
		var msg models.Message
		if err := rows.Scan(
			&msg.ChatID,
			&msg.MessageID,
			&msg.ThreadID,
			&msg.Content,
			&msg.MessageTime,
			&msg.ChatName,
			&msg.ChatDescription,
			&msg.SenderID,
			&msg.SenderName,
			&msg.AccountID,
			&msg.CreatedAt,
			&msg.Domain,
			&msg.Entities,
			&msg.Intent,
			&msg.IsAIHandled,
		); err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		messages = append(messages, msg)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return messages, nil
}

func (r *repository) UpdateSemantic(ctx context.Context, messageID string, chatID string, threadID string, domain []string, entities []string, intent []string) error {
	const op = "repositories.messages.UpdateSemantic"

	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel:   pgx.ReadCommitted,
		AccessMode: pgx.ReadWrite,
	})
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		UPDATE messages 
		SET domain = $1, 
			entities = $2, 
			intent = $3, 
			is_ai_handled = $4
		WHERE message_id = $5 AND chat_id = $6 AND thread_id = $7
	`, domain, entities, intent, true, messageID, chatID, threadID)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (r *repository) FindHandledByAI(ctx context.Context) ([]models.Message, error) {
	const op = "repositories.messages.FindHandledByAI"

	rows, err := r.pool.Query(ctx, `
	
SELECT 
    m.chat_id,
    m.message_id,
    m.thread_id,
    m.content,
    m.message_time,
    m.chat_name,
    m.chat_description,
    m.sender_id,
    m.sender_name,
    m.account_id,
    m.created_at,
    m.domain,
    m.entities,
    m.intent,
    m.is_ai_handled
FROM messages m
WHERE m.is_ai_handled = true 
    AND m.content != ''
	`)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var messages []models.Message
	for rows.Next() {
		var msg models.Message
		if err := rows.Scan(
			&msg.ChatID,
			&msg.MessageID,
			&msg.ThreadID,
			&msg.Content,
			&msg.MessageTime,
			&msg.ChatName,
			&msg.ChatDescription,
			&msg.SenderID,
			&msg.SenderName,
			&msg.AccountID,
			&msg.CreatedAt,
			&msg.Domain,
			&msg.Entities,
			&msg.Intent,
			&msg.IsAIHandled,
		); err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		messages = append(messages, msg)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return messages, nil
}
