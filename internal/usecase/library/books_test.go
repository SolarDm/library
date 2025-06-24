package library

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/project/library/generated/api/library"
	"github.com/project/library/generated/mocks"
	"github.com/project/library/internal/entity"
	"github.com/project/library/internal/usecase/repository"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func getDefaultBookUseCaseWithOutbox(
	ctrl *gomock.Controller,
	booksRepository *mocks.MockBooksRepository,
	transactor *mocks.MockTransactor,
	outboxRepository *mocks.MockOutboxRepository,
) *libraryImpl {
	authorRepo := mocks.NewMockAuthorRepository(ctrl)
	logger := zap.NewNop()

	return New(logger, transactor, outboxRepository, authorRepo, booksRepository)
}

func getDefaultBookUseCase(ctrl *gomock.Controller, booksRepository *mocks.MockBooksRepository) *libraryImpl {
	transactor := mocks.NewMockTransactor(ctrl)
	outboxRepo := mocks.NewMockOutboxRepository(ctrl)

	return getDefaultBookUseCaseWithOutbox(ctrl, booksRepository, transactor, outboxRepo)
}

func TestAddBook(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name             string
		request          *library.AddBookRequest
		expectedResponse *library.AddBookResponse
		repositoryError  error
		outboxError      error
		expectedError    error
	}{
		{
			name: "Run without errors",
			request: &library.AddBookRequest{
				Name:      "Test",
				AuthorIds: []string{"test"},
			},
			expectedResponse: &library.AddBookResponse{
				Book: &library.Book{
					Id:        "123",
					Name:      "Test",
					AuthorIds: []string{"test"},
					CreatedAt: timestamppb.New(time.Now()),
					UpdatedAt: timestamppb.New(time.Now()),
				},
			},
			repositoryError: nil,
			outboxError:     nil,
			expectedError:   nil,
		},
		{
			name: "Run with internal errors",
			request: &library.AddBookRequest{
				Name:      "Test",
				AuthorIds: []string{"test"},
			},
			expectedResponse: &library.AddBookResponse{
				Book: &library.Book{
					Id:        "123",
					Name:      "Test",
					AuthorIds: []string{"test"},
					CreatedAt: timestamppb.New(time.Now()),
					UpdatedAt: timestamppb.New(time.Now()),
				},
			},
			repositoryError: errors.New("test"),
			outboxError:     nil,
			expectedError:   status.Error(codes.Internal, "repository error"),
		},
		{
			name: "Run with not found errors",
			request: &library.AddBookRequest{
				Name:      "Test",
				AuthorIds: []string{"test"},
			},
			expectedResponse: &library.AddBookResponse{
				Book: &library.Book{
					Id:        "123",
					Name:      "Test",
					AuthorIds: []string{"test"},
					CreatedAt: timestamppb.New(time.Now()),
					UpdatedAt: timestamppb.New(time.Now()),
				},
			},
			repositoryError: entity.ErrAuthorNotFound,
			outboxError:     nil,
			expectedError:   status.Error(codes.NotFound, "author not found"),
		},
		{
			name: "Run with outbox errors",
			request: &library.AddBookRequest{
				Name:      "Test",
				AuthorIds: []string{"test"},
			},
			expectedResponse: &library.AddBookResponse{
				Book: &library.Book{
					Id:        "123",
					Name:      "Test",
					AuthorIds: []string{"test"},
					CreatedAt: timestamppb.New(time.Now()),
					UpdatedAt: timestamppb.New(time.Now()),
				},
			},
			repositoryError: nil,
			outboxError:     errors.New("outbox err"),
			expectedError:   status.Error(codes.Internal, "outbox err"),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			bookRepo := mocks.NewMockBooksRepository(ctrl)
			bookRepo.EXPECT().AddBook(gomock.Any(), gomock.Any()).
				Return(
					entity.Book{
						ID:        tc.expectedResponse.GetBook().GetId(),
						Name:      tc.expectedResponse.GetBook().GetName(),
						AuthorIDs: tc.expectedResponse.GetBook().GetAuthorIds(),
						CreatedAt: tc.expectedResponse.GetBook().GetCreatedAt().AsTime(),
						UpdatedAt: tc.expectedResponse.GetBook().GetUpdatedAt().AsTime(),
					},
					tc.repositoryError,
				)

			ctx := context.Background()

			transactor := mocks.NewMockTransactor(ctrl)
			transactor.EXPECT().WithTx(ctx, gomock.Any()).DoAndReturn(
				func(ctx context.Context, f func(ctx context.Context) error) error {
					return f(ctx)
				},
			)

			times := 0
			if tc.repositoryError == nil {
				times = 1
			}
			outboxRepo := mocks.NewMockOutboxRepository(ctrl)
			outboxRepo.EXPECT().SendMessage(ctx, repository.OutboxKindBook.String()+"_"+tc.expectedResponse.GetBook().GetId(),
				repository.OutboxKindBook, gomock.Any()).Return(tc.outboxError).Times(times)

			uc := getDefaultBookUseCaseWithOutbox(ctrl, bookRepo, transactor, outboxRepo)
			resp, err := uc.AddBook(ctx, tc.request)
			s, ok := status.FromError(err)
			expS, expOk := status.FromError(tc.expectedError)
			require.Equal(t, expOk, ok)
			if ok {
				require.Equal(t, expS.Code(), s.Code())
			} else {
				require.Equal(t, tc.expectedResponse, resp)
			}
		})
	}
}

func TestUpdateBook(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name             string
		request          *library.UpdateBookRequest
		expectedResponse *library.UpdateBookResponse
		repositoryError  error
		expectedError    error
	}{
		{
			name: "Run without errors",
			request: &library.UpdateBookRequest{
				Id:        uuid.NewString(),
				Name:      "Test",
				AuthorIds: []string{"test"},
			},
			expectedResponse: &library.UpdateBookResponse{},
			repositoryError:  nil,
			expectedError:    nil,
		},
		{
			name: "Run with internal errors",
			request: &library.UpdateBookRequest{
				Id:        uuid.NewString(),
				Name:      "Test",
				AuthorIds: []string{"test"},
			},
			expectedResponse: &library.UpdateBookResponse{},
			repositoryError:  errors.New("test error"),
			expectedError:    status.Error(codes.Internal, "repository error"),
		},
		{
			name: "Run with not found errors",
			request: &library.UpdateBookRequest{
				Id:        uuid.NewString(),
				Name:      "Test",
				AuthorIds: []string{"test"},
			},
			expectedResponse: &library.UpdateBookResponse{},
			repositoryError:  entity.ErrBookNotFound,
			expectedError:    status.Error(codes.NotFound, "book not found"),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			ctx := context.Background()
			bookRepo := mocks.NewMockBooksRepository(ctrl)
			bookRepo.EXPECT().UpdateBook(ctx, tc.request.GetId(), tc.request.GetName(), tc.request.GetAuthorIds()).
				Return(entity.Book{}, tc.repositoryError).AnyTimes()

			uc := getDefaultBookUseCase(ctrl, bookRepo)
			_, err := uc.UpdateBook(ctx, tc.request)
			s, ok := status.FromError(err)
			expS, expOk := status.FromError(tc.expectedError)
			require.Equal(t, expOk, ok)
			if ok {
				require.Equal(t, expS.Code(), s.Code())
			}
		})
	}
}

func TestGetBookInfo(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name             string
		request          *library.GetBookInfoRequest
		expectedResponse *library.GetBookInfoResponse
		repositoryError  error
		expectedError    error
	}{
		{
			name:    "Run without errors",
			request: &library.GetBookInfoRequest{Id: "123"},
			expectedResponse: &library.GetBookInfoResponse{Book: &library.Book{
				Id:        "123",
				Name:      "Test",
				AuthorIds: []string{"test"},
				CreatedAt: timestamppb.New(time.Now()),
				UpdatedAt: timestamppb.New(time.Now()),
			}},
			repositoryError: nil,
			expectedError:   nil,
		},
		{
			name:    "Run with internal errors",
			request: &library.GetBookInfoRequest{Id: "123"},
			expectedResponse: &library.GetBookInfoResponse{Book: &library.Book{
				Id:        "123",
				Name:      "test",
				AuthorIds: nil,
			}},
			repositoryError: errors.New("test error"),
			expectedError:   status.Error(codes.Internal, "repository error"),
		},
		{
			name:    "Run with not found errors",
			request: &library.GetBookInfoRequest{Id: "123"},
			expectedResponse: &library.GetBookInfoResponse{Book: &library.Book{
				Id:        "123",
				Name:      "test",
				AuthorIds: nil,
			}},
			repositoryError: entity.ErrBookNotFound,
			expectedError:   status.Error(codes.NotFound, "book not found"),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			ctx := context.Background()
			bookRepo := mocks.NewMockBooksRepository(ctrl)
			bookRepo.EXPECT().GetBookInfo(ctx, tc.request.GetId()).Return(
				entity.Book{
					ID:        tc.expectedResponse.GetBook().GetId(),
					Name:      tc.expectedResponse.GetBook().GetName(),
					AuthorIDs: tc.expectedResponse.GetBook().GetAuthorIds(),
					CreatedAt: tc.expectedResponse.GetBook().GetCreatedAt().AsTime(),
					UpdatedAt: tc.expectedResponse.GetBook().GetUpdatedAt().AsTime(),
				},
				tc.repositoryError,
			)

			uc := getDefaultBookUseCase(ctrl, bookRepo)
			resp, err := uc.GetBookInfo(ctx, tc.request)
			s, ok := status.FromError(err)
			expS, expOk := status.FromError(tc.expectedError)
			require.Equal(t, expOk, ok)
			if ok {
				require.Equal(t, expS.Code(), s.Code())
			} else {
				require.Equal(t, tc.expectedResponse, resp)
			}
		})
	}
}
