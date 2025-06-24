package controller

//go:generate ../../bin/mockgen --build_flags=--mod=mod -destination=../../generated/mocks/server_mock.go -package=mocks . GetAuthorBooksServer

import (
	generated "github.com/project/library/generated/api/library"
	"github.com/project/library/internal/usecase/library"
	"go.uber.org/zap"
)

type GetAuthorBooksServer interface {
	generated.Library_GetAuthorBooksServer
}

var _ generated.LibraryServer = (*implementation)(nil)

type implementation struct {
	logger        *zap.Logger
	booksUseCase  library.BooksUseCase
	authorUseCase library.AuthorUseCase
}

func New(
	logger *zap.Logger,
	booksUseCase library.BooksUseCase,
	authorUseCase library.AuthorUseCase,
) *implementation {
	return &implementation{
		logger:        logger,
		booksUseCase:  booksUseCase,
		authorUseCase: authorUseCase,
	}
}
