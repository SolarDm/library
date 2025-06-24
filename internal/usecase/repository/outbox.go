package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var _ OutboxRepository = (*outboxRepository)(nil)

type outboxRepository struct {
	db *pgxpool.Pool
}

func NewOutbox(db *pgxpool.Pool) *outboxRepository {
	return &outboxRepository{
		db: db,
	}
}

func (o *outboxRepository) SendMessage(ctx context.Context, idempotencyKey string, kind OutboxKind, message []byte) error {
	const query = `
INSERT INTO outbox (idempotency_key, data, status, kind)
VALUES($1, $2, 'CREATED', $3)
ON CONFLICT (idempotency_key) DO NOTHING`

	var err error
	if tx, txErr := extractTx(ctx); txErr == nil {
		_, err = tx.Exec(ctx, query, idempotencyKey, message, kind)
	} else {
		_, err = o.db.Exec(ctx, query, idempotencyKey, message, kind)
	}

	if err != nil {
		return err
	}

	return nil
}

func (o *outboxRepository) GetMessages(ctx context.Context, batchSize int, inProgressTTL time.Duration) ([]OutboxData, error) {
	const query = `
UPDATE outbox
SET status = 'IN_PROGRESS'
WHERE idempotency_key IN (
    SELECT idempotency_key
    FROM outbox
    WHERE
        (status = 'CREATED'
            OR (status = 'IN_PROGRESS' AND updated_at < now() - $1::interval))
    ORDER BY created_at
    LIMIT $2
    FOR UPDATE SKIP LOCKED
	)
	RETURNING idempotency_key, data, kind;`

	internal := fmt.Sprintf("%d ms", inProgressTTL.Milliseconds())

	var (
		err  error
		rows pgx.Rows
	)
	if tx, txErr := extractTx(ctx); txErr == nil {
		rows, err = tx.Query(ctx, query, internal, batchSize)
	} else {
		rows, err = o.db.Query(ctx, query, internal, batchSize)
	}

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	result := make([]OutboxData, 0)

	for rows.Next() {
		var key string
		var rawData []byte
		var kind OutboxKind

		if err := rows.Scan(&key, &rawData, &kind); err != nil {
			return nil, err
		}

		result = append(result, OutboxData{
			IdempotencyKey: key,
			RawData:        rawData,
			Kind:           kind,
		})
	}

	return result, rows.Err()
}

func (o *outboxRepository) MarkAsProcessed(ctx context.Context, idempotencyKeys []string) error {
	if len(idempotencyKeys) == 0 {
		return nil
	}

	const query = `
UPDATE outbox
SET status = 'SUCCESS'
WHERE idempotency_key = ANY($1);
`

	var err error
	if tx, txErr := extractTx(ctx); txErr == nil {
		_, err = tx.Exec(ctx, query, idempotencyKeys)
	} else {
		_, err = o.db.Exec(ctx, query, idempotencyKeys)
	}

	if err != nil {
		return err
	}

	return nil
}
