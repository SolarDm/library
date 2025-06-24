package outbox

import (
	"context"
	"errors"
	"math/rand/v2"
	"slices"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/project/library/config"
	"github.com/project/library/generated/mocks"
	"github.com/project/library/internal/usecase/repository"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

func TestOutbox(t *testing.T) {
	t.Parallel()

	type arguments struct {
		workers       int
		batchSize     int
		waitTime      time.Duration
		inProgressTTL time.Duration
	}

	testCases := []struct {
		name          string
		args          arguments
		messagesCount int
		outboxEnabled bool
		waitTime      time.Duration
	}{
		{
			name: "no outbox",
			args: arguments{
				workers:       1,
				batchSize:     1,
				waitTime:      1 * time.Millisecond,
				inProgressTTL: 1 * time.Millisecond,
			},
			messagesCount: 10,
			outboxEnabled: false,
			waitTime:      1 * time.Second,
		},
		{
			name: "run one worker",
			args: arguments{
				workers:       1,
				batchSize:     1,
				waitTime:      1 * time.Millisecond,
				inProgressTTL: 1 * time.Millisecond,
			},
			messagesCount: 10,
			outboxEnabled: true,
			waitTime:      1 * time.Second,
		},
		{
			name: "run multiple workers",
			args: arguments{
				workers:       10,
				batchSize:     5,
				waitTime:      1 * time.Millisecond,
				inProgressTTL: 1 * time.Millisecond,
			},
			messagesCount: 100,
			outboxEnabled: true,
			waitTime:      1 * time.Second,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			logger := zap.NewNop()
			outboxRepo := mocks.NewMockOutboxRepository(ctrl)
			transactor := mocks.NewMockTransactor(ctrl)
			cfg := &config.Config{
				Outbox: config.Outbox{
					Enabled: tc.outboxEnabled,
				},
			}
			ctx, cancel := context.WithCancel(context.Background())

			transactor.EXPECT().WithTx(ctx, gomock.Any()).DoAndReturn(
				func(_ context.Context, f func(ctx context.Context) error) error {
					return f(ctx)
				},
			).AnyTimes()

			need := tc.messagesCount
			mx := &sync.Mutex{}

			givenKeys := make([]string, 0)
			givenData := make([][]byte, 0)
			givenKinds := make([]repository.OutboxKind, 0)

			gottenKeys := make([]string, 0)
			gottenData := make([][]byte, 0)
			gottenKinds := make([]repository.OutboxKind, 0)

			outboxRepo.EXPECT().GetMessages(ctx, tc.args.batchSize, tc.args.inProgressTTL).DoAndReturn(
				func(ctx context.Context, batchSize int, inProgressTTL time.Duration) ([]repository.OutboxData, error) {
					if rand.Int()%2 == 1 {
						return nil, errors.New("test")
					}

					mx.Lock()
					defer mx.Unlock()

					data := make([]repository.OutboxData, 0)
					for i := 0; i < min(need, batchSize); i++ {
						data = append(data, repository.OutboxData{
							IdempotencyKey: strconv.Itoa(i),
							Kind:           repository.OutboxKindBook,
							RawData:        []byte{byte(i)},
						})
						givenKeys = append(givenKeys, strconv.Itoa(i))
						givenData = append(givenData, []byte{byte(i)})
						givenKinds = append(givenKinds, repository.OutboxKindBook)
					}

					data = append(data, repository.OutboxData{
						IdempotencyKey: strconv.Itoa(-1),
						Kind:           repository.OutboxKindUndefined,
						RawData:        []byte{byte(0)},
					})

					data = append(data, repository.OutboxData{
						IdempotencyKey: strconv.Itoa(-1),
						Kind:           repository.OutboxKindBook,
						RawData:        nil,
					})

					need -= len(data)

					return data, nil
				},
			).AnyTimes()

			outboxRepo.EXPECT().MarkAsProcessed(ctx, gomock.Any()).DoAndReturn(
				func(ctx context.Context, idempotencyKeys []string) error {
					mx.Lock()
					defer mx.Unlock()

					gottenKeys = append(gottenKeys, idempotencyKeys...)
					if rand.Int()%2 == 1 {
						return errors.New("test")
					}
					return nil
				},
			).AnyTimes()

			globalHandler := func(kind repository.OutboxKind) (KindHandler, error) {
				mx.Lock()
				defer mx.Unlock()

				if kind == repository.OutboxKindUndefined {
					return nil, errors.New("test")
				}
				gottenKinds = append(gottenKinds, kind)

				return func(_ context.Context, data []byte) error {
					mx.Lock()
					defer mx.Unlock()

					if data == nil {
						for i, k := range gottenKinds {
							if k == kind {
								gottenKinds = append(gottenKinds[:i], gottenKinds[i+1:]...)
								break
							}
						}
						return errors.New("test")
					}

					gottenData = append(gottenData, data)
					return nil
				}, nil
			}

			outbox := New(logger, outboxRepo, globalHandler, cfg, transactor)

			go outbox.Start(ctx, tc.args.workers, tc.args.batchSize, tc.args.waitTime, tc.args.inProgressTTL)

			time.Sleep(tc.waitTime)
			cancel()

			mx.Lock()
			defer mx.Unlock()

			slices.Sort(givenKinds)
			slices.Sort(givenKeys)

			slices.Sort(gottenKinds)
			slices.Sort(gottenKeys)

			require.Equal(t, len(givenKinds), len(gottenKinds))
			require.Equal(t, givenKeys, gottenKeys)
			require.Equal(t, len(givenData), len(gottenData))
		})
	}
}
