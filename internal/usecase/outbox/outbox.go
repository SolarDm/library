package outbox

import (
	"context"
	"sync"
	"time"

	"github.com/project/library/config"
	"github.com/project/library/internal/usecase/repository"
	"go.uber.org/zap"
)

type GlobalHandler = func(kind repository.OutboxKind) (KindHandler, error)
type KindHandler = func(ctx context.Context, data []byte) error

type Outbox interface {
	Start(ctx context.Context, workers int, batchSize int, waitTime time.Duration, inProgressTTL time.Duration)
}

var _ Outbox = (*outboxImpl)(nil)

type outboxImpl struct {
	logger           *zap.Logger
	outboxRepository repository.OutboxRepository
	globalHandler    GlobalHandler
	cfg              *config.Config
	transactor       repository.Transactor
}

func New(
	logger *zap.Logger,
	outboxRepository repository.OutboxRepository,
	globalHandler GlobalHandler,
	cfg *config.Config,
	transactor repository.Transactor,
) *outboxImpl {
	return &outboxImpl{
		logger:           logger,
		outboxRepository: outboxRepository,
		globalHandler:    globalHandler,
		cfg:              cfg,
		transactor:       transactor,
	}
}

func (o *outboxImpl) Start(
	ctx context.Context,
	workers int,
	batchSize int,
	waitTime time.Duration,
	inProgressTTL time.Duration,
) {
	wg := new(sync.WaitGroup)

	for workerID := 1; workerID <= workers; workerID++ {
		wg.Add(1)
		go o.worker(ctx, wg, batchSize, waitTime, inProgressTTL)
	}

	wg.Wait()
}

func (o *outboxImpl) worker(
	ctx context.Context,
	wg *sync.WaitGroup,
	batchSize int,
	waitTIme time.Duration,
	inProgressTTL time.Duration,
) {
	defer wg.Done()

	for {
		time.Sleep(waitTIme)

		if !o.cfg.Outbox.Enabled {
			continue
		}

		err := o.transactor.WithTx(ctx, func(ctx context.Context) error {
			messages, err := o.outboxRepository.GetMessages(ctx, batchSize, inProgressTTL)

			if err != nil {
				o.logger.Error("can not fetch messages from outbox", zap.Error(err))
				return err
			}

			o.logger.Info("messages fetched", zap.Int("size", len(messages)))
			successKeys := make([]string, 0, len(messages))

			for i := 0; i < len(messages); i++ {
				message := messages[i]
				key := message.IdempotencyKey

				kindHandler, handleErr := o.globalHandler(message.Kind)

				if handleErr != nil {
					o.logger.Error("unexpected kind", zap.Error(err))
					continue
				}

				err = kindHandler(ctx, message.RawData)

				if err != nil {
					o.logger.Error("kind error", zap.Error(err))
					continue
				}

				successKeys = append(successKeys, key)
			}

			err = o.outboxRepository.MarkAsProcessed(ctx, successKeys)
			if err != nil {
				o.logger.Error("mark as processed outbox error", zap.Error(err))
				return err
			}

			return nil
		})

		if err != nil {
			o.logger.Error("worker stage error", zap.Error(err))
		}

		select {
		case <-ctx.Done():
			return
		default:
			continue
		}
	}
}
