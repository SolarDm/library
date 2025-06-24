package controller

import (
	"testing"
	"time"

	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/google/uuid"
	"github.com/project/library/generated/api/library"
	"github.com/project/library/generated/mocks"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestAddBook(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name             string
		request          *library.AddBookRequest
		expectedResponse *library.AddBookResponse
		expectedError    error
	}{
		{
			name: "No error",
			request: &library.AddBookRequest{
				Name:      "test",
				AuthorIds: []string{uuid.NewString(), uuid.NewString(), uuid.NewString()},
			},
			expectedResponse: &library.AddBookResponse{Book: &library.Book{
				Id:        uuid.NewString(),
				Name:      "test",
				AuthorIds: []string{uuid.NewString(), uuid.NewString(), uuid.NewString()},
				CreatedAt: timestamppb.New(time.Now()),
				UpdatedAt: timestamppb.New(time.Now()),
			}},
			expectedError: nil,
		},
		{
			name: "Id validation error",
			request: &library.AddBookRequest{
				Name:      "test",
				AuthorIds: []string{"1"},
			},
			expectedResponse: &library.AddBookResponse{Book: &library.Book{}},
			expectedError:    status.Error(codes.InvalidArgument, "test"),
		},
		{
			name:             "Internal error",
			request:          &library.AddBookRequest{},
			expectedResponse: &library.AddBookResponse{Book: &library.Book{}},
			expectedError:    status.Error(codes.Internal, "test"),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			booksUseCase := mocks.NewMockBooksUseCase(ctrl)
			booksUseCase.EXPECT().AddBook(gomock.Any(), tc.request).
				Return(tc.expectedResponse, tc.expectedError).AnyTimes()

			logger := zap.NewNop()
			authorUseCase := mocks.NewMockAuthorUseCase(ctrl)
			service := New(logger, booksUseCase, authorUseCase)

			ctx := context.Background()
			response, err := service.AddBook(ctx, tc.request)

			if tc.expectedError != nil {
				s, ok := status.FromError(err)
				expS, expOk := status.FromError(tc.expectedError)
				require.Equal(t, expOk, ok)
				if ok {
					require.Equal(t, s.Code(), expS.Code())
				}
			} else {
				require.Equal(t, tc.expectedResponse, response)
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
		expectedError    error
	}{
		{
			name: "No error",
			request: &library.ChangeAuthorInfoRequest{
				Id:   uuid.NewString(),
				Name: "test",
			},
			expectedResponse: &library.ChangeAuthorInfoResponse{},
			expectedError:    nil,
		},
		{
			name: "Id validation error",
			request: &library.ChangeAuthorInfoRequest{
				Id:   "1",
				Name: "test",
			},
			expectedResponse: &library.ChangeAuthorInfoResponse{},
			expectedError:    status.Error(codes.InvalidArgument, "test"),
		},
		{
			name: "Name validation error",
			request: &library.ChangeAuthorInfoRequest{
				Id:   uuid.NewString(),
				Name: "\\+*",
			},
			expectedResponse: &library.ChangeAuthorInfoResponse{},
			expectedError:    status.Error(codes.InvalidArgument, "test"),
		},
		{
			name: "Internal error",
			request: &library.ChangeAuthorInfoRequest{
				Id:   uuid.NewString(),
				Name: "test",
			},
			expectedResponse: &library.ChangeAuthorInfoResponse{},
			expectedError:    status.Error(codes.Internal, "test"),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			authorUseCase := mocks.NewMockAuthorUseCase(ctrl)
			authorUseCase.EXPECT().ChangeAuthorInfo(gomock.Any(), tc.request).
				Return(tc.expectedResponse, tc.expectedError).AnyTimes()

			logger := zap.NewNop()
			booksUseCase := mocks.NewMockBooksUseCase(ctrl)
			service := New(logger, booksUseCase, authorUseCase)

			ctx := context.Background()
			response, err := service.ChangeAuthorInfo(ctx, tc.request)

			if tc.expectedError != nil {
				s, ok := status.FromError(err)
				expS, expOk := status.FromError(tc.expectedError)
				require.Equal(t, expOk, ok)
				if ok {
					require.Equal(t, s.Code(), expS.Code())
				}
			} else {
				require.Equal(t, tc.expectedResponse, response)
			}
		})
	}
}

func TestGetAuthorBooks(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		request       *library.GetAuthorBooksRequest
		expectedError error
	}{
		{
			name: "No error",
			request: &library.GetAuthorBooksRequest{
				AuthorId: uuid.NewString(),
			},
			expectedError: nil,
		},
		{
			name: "Id validation error",
			request: &library.GetAuthorBooksRequest{
				AuthorId: "1",
			},
			expectedError: status.Error(codes.InvalidArgument, "test"),
		},
		{
			name: "Author not found error",
			request: &library.GetAuthorBooksRequest{
				AuthorId: uuid.NewString(),
			},
			expectedError: status.Error(codes.NotFound, "test"),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			server := mocks.NewMockGetAuthorBooksServer(ctrl)
			server.EXPECT().Context().Return(context.Background()).AnyTimes()

			authorUseCase := mocks.NewMockAuthorUseCase(ctrl)
			authorUseCase.EXPECT().GetAuthorBooks(gomock.Any(), tc.request, server).
				Return(tc.expectedError).AnyTimes()

			logger := zap.NewNop()
			booksUseCase := mocks.NewMockBooksUseCase(ctrl)
			service := New(logger, booksUseCase, authorUseCase)

			err := service.GetAuthorBooks(tc.request, server)

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
		expectedError    error
	}{
		{
			name: "No error",
			request: &library.GetAuthorInfoRequest{
				Id: uuid.NewString(),
			},
			expectedResponse: &library.GetAuthorInfoResponse{
				Id:   uuid.NewString(),
				Name: "test",
			},
			expectedError: nil,
		},
		{
			name: "Id validation error",
			request: &library.GetAuthorInfoRequest{
				Id: "1",
			},
			expectedResponse: &library.GetAuthorInfoResponse{
				Id:   uuid.NewString(),
				Name: "test",
			},
			expectedError: status.Error(codes.InvalidArgument, "test"),
		},
		{
			name: "Internal error",
			request: &library.GetAuthorInfoRequest{
				Id: uuid.NewString(),
			},
			expectedResponse: &library.GetAuthorInfoResponse{
				Id:   uuid.NewString(),
				Name: "test",
			},
			expectedError: status.Error(codes.Internal, "test"),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			authorUseCase := mocks.NewMockAuthorUseCase(ctrl)
			authorUseCase.EXPECT().GetAuthorInfo(gomock.Any(), tc.request).
				Return(tc.expectedResponse, tc.expectedError).AnyTimes()

			logger := zap.NewNop()
			booksUseCase := mocks.NewMockBooksUseCase(ctrl)
			service := New(logger, booksUseCase, authorUseCase)

			ctx := context.Background()
			response, err := service.GetAuthorInfo(ctx, tc.request)

			if tc.expectedError != nil {
				s, ok := status.FromError(err)
				expS, expOk := status.FromError(tc.expectedError)
				require.Equal(t, expOk, ok)
				if ok {
					require.Equal(t, s.Code(), expS.Code())
				}
			} else {
				require.Equal(t, tc.expectedResponse, response)
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
		expectedError    error
	}{
		{
			name: "No error",
			request: &library.GetBookInfoRequest{
				Id: uuid.NewString(),
			},
			expectedResponse: &library.GetBookInfoResponse{Book: &library.Book{
				Id:        uuid.NewString(),
				Name:      "test",
				AuthorIds: []string{uuid.NewString(), uuid.NewString(), uuid.NewString()},
				CreatedAt: timestamppb.New(time.Now()),
				UpdatedAt: timestamppb.New(time.Now()),
			}},
			expectedError: nil,
		},
		{
			name: "Id validation error",
			request: &library.GetBookInfoRequest{
				Id: "12",
			},
			expectedResponse: &library.GetBookInfoResponse{Book: &library.Book{}},
			expectedError:    status.Error(codes.InvalidArgument, "test"),
		},
		{
			name: "Internal error",
			request: &library.GetBookInfoRequest{
				Id: uuid.NewString(),
			},
			expectedResponse: &library.GetBookInfoResponse{Book: &library.Book{}},
			expectedError:    status.Error(codes.Internal, "test"),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			booksUseCase := mocks.NewMockBooksUseCase(ctrl)
			booksUseCase.EXPECT().GetBookInfo(gomock.Any(), tc.request).
				Return(tc.expectedResponse, tc.expectedError).AnyTimes()

			logger := zap.NewNop()
			authorUseCase := mocks.NewMockAuthorUseCase(ctrl)
			service := New(logger, booksUseCase, authorUseCase)

			ctx := context.Background()
			response, err := service.GetBookInfo(ctx, tc.request)

			if tc.expectedError != nil {
				s, ok := status.FromError(err)
				expS, expOk := status.FromError(tc.expectedError)
				require.Equal(t, expOk, ok)
				if ok {
					require.Equal(t, s.Code(), expS.Code())
				}
			} else {
				require.Equal(t, tc.expectedResponse, response)
			}
		})
	}
}

func TestRegisterAuthor(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name             string
		request          *library.RegisterAuthorRequest
		expectedResponse *library.RegisterAuthorResponse
		expectedError    error
	}{
		{
			name: "No error",
			request: &library.RegisterAuthorRequest{
				Name: "test",
			},
			expectedResponse: &library.RegisterAuthorResponse{
				Id: uuid.NewString(),
			},
			expectedError: nil,
		},
		{
			name: "Name validation error",
			request: &library.RegisterAuthorRequest{
				Name: "\\*/j",
			},
			expectedResponse: &library.RegisterAuthorResponse{
				Id: uuid.NewString(),
			},
			expectedError: status.Error(codes.InvalidArgument, "test"),
		},
		{
			name: "Internal error",
			request: &library.RegisterAuthorRequest{
				Name: "test",
			},
			expectedResponse: &library.RegisterAuthorResponse{
				Id: uuid.NewString(),
			},
			expectedError: status.Error(codes.Internal, "test"),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			authorUseCase := mocks.NewMockAuthorUseCase(ctrl)
			authorUseCase.EXPECT().RegisterAuthor(gomock.Any(), tc.request).
				Return(tc.expectedResponse, tc.expectedError).AnyTimes()

			logger := zap.NewNop()
			booksUseCase := mocks.NewMockBooksUseCase(ctrl)
			service := New(logger, booksUseCase, authorUseCase)

			ctx := context.Background()
			response, err := service.RegisterAuthor(ctx, tc.request)

			if tc.expectedError != nil {
				s, ok := status.FromError(err)
				expS, expOk := status.FromError(tc.expectedError)
				require.Equal(t, expOk, ok)
				if ok {
					require.Equal(t, s.Code(), expS.Code())
				}
			} else {
				require.Equal(t, tc.expectedResponse, response)
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
		expectedError    error
	}{
		{
			name: "No error",
			request: &library.UpdateBookRequest{
				Id:        uuid.NewString(),
				Name:      "test",
				AuthorIds: []string{uuid.NewString(), uuid.NewString(), uuid.NewString()},
			},
			expectedResponse: &library.UpdateBookResponse{},
			expectedError:    nil,
		},
		{
			name: "Book id validation error",
			request: &library.UpdateBookRequest{
				Id:        "5",
				Name:      "test",
				AuthorIds: []string{uuid.NewString(), uuid.NewString(), uuid.NewString()},
			},
			expectedResponse: &library.UpdateBookResponse{},
			expectedError:    status.Error(codes.InvalidArgument, "test"),
		},
		{
			name: "Author id validation error",
			request: &library.UpdateBookRequest{
				Id:        uuid.NewString(),
				Name:      "test",
				AuthorIds: []string{uuid.NewString(), "5", uuid.NewString()},
			},
			expectedResponse: &library.UpdateBookResponse{},
			expectedError:    status.Error(codes.InvalidArgument, "test"),
		},
		{
			name: "Internal error",
			request: &library.UpdateBookRequest{
				Id:        uuid.NewString(),
				Name:      "test",
				AuthorIds: []string{uuid.NewString(), uuid.NewString(), uuid.NewString()},
			},
			expectedResponse: &library.UpdateBookResponse{},
			expectedError:    status.Error(codes.Internal, "test"),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			booksUseCase := mocks.NewMockBooksUseCase(ctrl)
			booksUseCase.EXPECT().UpdateBook(gomock.Any(), tc.request).Return(tc.expectedResponse, tc.expectedError).AnyTimes()

			logger := zap.NewNop()
			authorUseCase := mocks.NewMockAuthorUseCase(ctrl)
			service := New(logger, booksUseCase, authorUseCase)

			ctx := context.Background()
			response, err := service.UpdateBook(ctx, tc.request)

			if tc.expectedError != nil {
				s, ok := status.FromError(err)
				expS, expOk := status.FromError(tc.expectedError)
				require.Equal(t, expOk, ok)
				if ok {
					require.Equal(t, s.Code(), expS.Code())
				}
			} else {
				require.Equal(t, tc.expectedResponse, response)
			}
		})
	}
}
