package library

import (
	"context"
	"errors"
	"math/rand/v2"
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
)

func getDefaultAuthorUseCaseWithOutbox(
	ctrl *gomock.Controller,
	authorsRepository *mocks.MockAuthorRepository,
	transactor *mocks.MockTransactor,
	outboxRepository *mocks.MockOutboxRepository,
) *libraryImpl {
	booksRepo := mocks.NewMockBooksRepository(ctrl)
	logger := zap.NewNop()

	return New(logger, transactor, outboxRepository, authorsRepository, booksRepo)
}

func getDefaultAuthorUseCase(ctrl *gomock.Controller, authorsRepository *mocks.MockAuthorRepository) *libraryImpl {
	transactor := mocks.NewMockTransactor(ctrl)
	outboxRepo := mocks.NewMockOutboxRepository(ctrl)

	return getDefaultAuthorUseCaseWithOutbox(ctrl, authorsRepository, transactor, outboxRepo)
}

func TestRegisterAuthor(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name             string
		request          *library.RegisterAuthorRequest
		expectedResponse *library.RegisterAuthorResponse
		repositoryError  error
		outboxError      error
		expectedError    error
	}{
		{
			name: "Run without errors",
			request: &library.RegisterAuthorRequest{
				Name: "Test",
			},
			expectedResponse: &library.RegisterAuthorResponse{},
			repositoryError:  nil,
			expectedError:    nil,
		},
		{
			name: "Run with internal errors",
			request: &library.RegisterAuthorRequest{
				Name: "Test",
			},
			expectedResponse: &library.RegisterAuthorResponse{
				Id: uuid.NewString(),
			},
			repositoryError: errors.New("test"),
			outboxError:     nil,
			expectedError:   status.Error(codes.Internal, "repository error"),
		},
		{
			name: "Run with outbox errors",
			request: &library.RegisterAuthorRequest{
				Name: "Test",
			},
			expectedResponse: &library.RegisterAuthorResponse{
				Id: uuid.NewString(),
			},
			repositoryError: nil,
			outboxError:     errors.New("test"),
			expectedError:   status.Error(codes.Internal, "repository error"),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			ctx := context.Background()
			authorRepo := mocks.NewMockAuthorRepository(ctrl)
			authorRepo.EXPECT().RegisterAuthor(ctx, entity.Author{Name: tc.request.GetName()}).
				Return(
					entity.Author{
						ID:   tc.expectedResponse.GetId(),
						Name: tc.request.GetName(),
					},
					tc.repositoryError,
				)

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
			outboxRepo.EXPECT().SendMessage(ctx, repository.OutboxKindAuthor.String()+"_"+tc.expectedResponse.GetId(),
				repository.OutboxKindAuthor, gomock.Any()).Return(tc.outboxError).Times(times)

			uc := getDefaultAuthorUseCaseWithOutbox(ctrl, authorRepo, transactor, outboxRepo)
			_, err := uc.RegisterAuthor(ctx, tc.request)
			s, ok := status.FromError(err)
			expS, expOk := status.FromError(tc.expectedError)
			require.Equal(t, expOk, ok)
			if ok {
				require.Equal(t, s.Code(), expS.Code())
			}
		})
	}
}

func TestChangeAuthorInfo(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name             string
		request          *library.ChangeAuthorInfoRequest
		expectedResponse *library.ChangeAuthorInfoResponse
		repositoryError  error
		expectedError    error
	}{
		{
			name: "Run without errors",
			request: &library.ChangeAuthorInfoRequest{
				Id:   uuid.NewString(),
				Name: "test",
			},
			expectedResponse: &library.ChangeAuthorInfoResponse{},
			repositoryError:  nil,
			expectedError:    nil,
		},
		{
			name: "Run with internal errors",
			request: &library.ChangeAuthorInfoRequest{
				Id:   uuid.NewString(),
				Name: "test",
			},
			expectedResponse: &library.ChangeAuthorInfoResponse{},
			repositoryError:  errors.New("test error"),
			expectedError:    status.Error(codes.Internal, "repository error"),
		},
		{
			name: "Run with not found errors",
			request: &library.ChangeAuthorInfoRequest{
				Id:   uuid.NewString(),
				Name: "test",
			},
			expectedResponse: &library.ChangeAuthorInfoResponse{},
			repositoryError:  entity.ErrAuthorNotFound,
			expectedError:    status.Error(codes.NotFound, "author not found"),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			ctx := context.Background()
			repo := mocks.NewMockAuthorRepository(ctrl)
			repo.EXPECT().ChangeAuthorInfo(ctx, tc.request.GetId(), tc.request.GetName()).
				Return(entity.Author{}, tc.repositoryError).AnyTimes()
			uc := getDefaultAuthorUseCase(ctrl, repo)

			_, err := uc.ChangeAuthorInfo(ctx, tc.request)
			s, ok := status.FromError(err)
			expS, expOk := status.FromError(tc.expectedError)
			require.Equal(t, expOk, ok)
			if ok {
				require.Equal(t, s.Code(), expS.Code())
			}
		})
	}
}

func TestGetAuthorInfo(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name             string
		request          *library.GetAuthorInfoRequest
		expectedResponse *library.GetAuthorInfoResponse
		repositoryError  error
		expectedError    error
	}{
		{
			name:    "Run without errors",
			request: &library.GetAuthorInfoRequest{Id: "123"},
			expectedResponse: &library.GetAuthorInfoResponse{
				Id:   "123",
				Name: "Test",
			},
			repositoryError: nil,
			expectedError:   nil,
		},
		{
			name:    "Run with internal errors",
			request: &library.GetAuthorInfoRequest{Id: "123"},
			expectedResponse: &library.GetAuthorInfoResponse{
				Id:   "123",
				Name: "Test",
			},
			repositoryError: errors.New("test error"),
			expectedError:   status.Error(codes.Internal, "repository error"),
		},
		{
			name:    "Run with not found errors",
			request: &library.GetAuthorInfoRequest{Id: "123"},
			expectedResponse: &library.GetAuthorInfoResponse{
				Id:   "123",
				Name: "Test",
			},
			repositoryError: entity.ErrAuthorNotFound,
			expectedError:   status.Error(codes.NotFound, "author not found"),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			ctx := context.Background()
			AuthorRepo := mocks.NewMockAuthorRepository(ctrl)
			AuthorRepo.EXPECT().GetAuthorInfo(ctx, tc.request.GetId()).
				Return(entity.Author{
					ID:   tc.expectedResponse.GetId(),
					Name: tc.expectedResponse.GetName(),
				}, tc.repositoryError)

			uc := getDefaultAuthorUseCase(ctrl, AuthorRepo)
			resp, err := uc.GetAuthorInfo(ctx, tc.request)
			s, ok := status.FromError(err)
			expS, expOk := status.FromError(tc.expectedError)
			require.Equal(t, expOk, ok)
			if ok {
				require.Equal(t, s.Code(), expS.Code())
			} else {
				require.Equal(t, tc.expectedResponse, resp)
			}
		})
	}
}

func TestGetAuthorBooks(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name             string
		request          *library.GetAuthorBooksRequest
		expectedResponse []entity.Book
		repositoryError  error
		expectedError    error
	}{
		{
			name: "Run without errors",
			request: &library.GetAuthorBooksRequest{
				AuthorId: "123",
			},
			expectedResponse: []entity.Book{
				{
					ID:        "123",
					Name:      "test",
					AuthorIDs: []string{"123"},
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				{
					ID:        "456",
					Name:      "test",
					AuthorIDs: []string{"123"},
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				{
					ID:        "789",
					Name:      "test",
					AuthorIDs: []string{"123"},
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
			},
			repositoryError: nil,
			expectedError:   nil,
		},
		{
			name: "Run with internal errors",
			request: &library.GetAuthorBooksRequest{
				AuthorId: "123",
			},
			expectedResponse: []entity.Book{
				{
					ID:        "432",
					Name:      "test",
					AuthorIDs: []string{"123"},
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
			},
			repositoryError: errors.New("test error"),
			expectedError:   status.Error(codes.Internal, "repository error"),
		},
		{
			name: "Run with not found errors",
			request: &library.GetAuthorBooksRequest{
				AuthorId: "123",
			},
			expectedResponse: []entity.Book{
				{
					ID:        "432",
					Name:      "test",
					AuthorIDs: []string{"123"},
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
			},
			repositoryError: entity.ErrAuthorNotFound,
			expectedError:   status.Error(codes.NotFound, "author not found"),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			ctx := context.Background()

			books := make([]*library.Book, 0)
			server := mocks.NewMockGetAuthorBooksServer(ctrl)
			server.EXPECT().Send(gomock.Any()).DoAndReturn(func(book *library.Book) error {
				books = append(books, book)
				if rand.Int()%2 == 1 {
					return errors.New("append error")
				}
				return nil
			}).AnyTimes()

			authorRepo := mocks.NewMockAuthorRepository(ctrl)

			entityBooks := make([]entity.Book, 0)
			for _, book := range tc.expectedResponse {
				entityBooks = append(entityBooks,
					entity.Book{
						ID:        book.ID,
						Name:      book.Name,
						AuthorIDs: book.AuthorIDs,
						CreatedAt: book.CreatedAt,
						UpdatedAt: book.UpdatedAt,
					},
				)
			}

			authorRepo.EXPECT().GetAuthorBooks(ctx, tc.request.GetAuthorId()).
				Return(entityBooks, tc.repositoryError)

			uc := getDefaultAuthorUseCase(ctrl, authorRepo)
			err := uc.GetAuthorBooks(ctx, tc.request, server)
			s, ok := status.FromError(err)
			expS, expOk := status.FromError(tc.expectedError)
			require.Equal(t, expOk, ok)
			if ok {
				require.Equal(t, expS.Code(), s.Code())
			} else {
				require.Equal(t, tc.expectedResponse, books)
			}
		})
	}
}
