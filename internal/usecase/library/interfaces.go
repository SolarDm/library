package library

//go:generate ../../../bin/mockgen --build_flags=--mod=mod -destination=../../../generated/mocks/use_case_mock.go -package=mocks . AuthorUseCase,BooksUseCase

import (
	"context"

	"github.com/project/library/generated/api/library"
	"github.com/project/library/internal/usecase/repository"
	"go.uber.org/zap"
)

type (
	AuthorUseCase interface {
		RegisterAuthor(ctx context.Context, request *library.RegisterAuthorRequest) (*library.RegisterAuthorResponse, error)
		ChangeAuthorInfo(ctx context.Context, request *library.ChangeAuthorInfoRequest) (*library.ChangeAuthorInfoResponse, error)
		GetAuthorInfo(ctx context.Context, request *library.GetAuthorInfoRequest) (*library.GetAuthorInfoResponse, error)
		GetAuthorBooks(ctx context.Context, request *library.GetAuthorBooksRequest, resp library.Library_GetAuthorBooksServer) error
	}

	BooksUseCase interface {
		AddBook(ctx context.Context, request *library.AddBookRequest) (*library.AddBookResponse, error)
		UpdateBook(ctx context.Context, request *library.UpdateBookRequest) (*library.UpdateBookResponse, error)
		GetBookInfo(ctx context.Context, request *library.GetBookInfoRequest) (*library.GetBookInfoResponse, error)
	}
)

var _ AuthorUseCase = (*libraryImpl)(nil)
var _ BooksUseCase = (*libraryImpl)(nil)

type libraryImpl struct {
	logger           *zap.Logger
	transactor       repository.Transactor
	outboxRepository repository.OutboxRepository
	authorRepository repository.AuthorRepository
	booksRepository  repository.BooksRepository
}

func New(
	logger *zap.Logger,
	transactor repository.Transactor,
	outboxRepository repository.OutboxRepository,
	authorRepository repository.AuthorRepository,
	booksRepository repository.BooksRepository,
) *libraryImpl {
	return &libraryImpl{
		logger:           logger,
		transactor:       transactor,
		outboxRepository: outboxRepository,
		authorRepository: authorRepository,
		booksRepository:  booksRepository,
	}
}
