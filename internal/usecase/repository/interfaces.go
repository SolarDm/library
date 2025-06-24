package repository

//go:generate ../../../bin/mockgen --build_flags=--mod=mod -destination=../../../generated/mocks/repository_mock.go -package=mocks . AuthorRepository,BooksRepository,Transactor,OutboxRepository

import (
	"context"
	"time"

	"github.com/project/library/internal/entity"
)

type (
	AuthorRepository interface {
		RegisterAuthor(ctx context.Context, author entity.Author) (entity.Author, error)
		ChangeAuthorInfo(ctx context.Context, id string, name string) (entity.Author, error)
		GetAuthorInfo(ctx context.Context, id string) (entity.Author, error)
		GetAuthorBooks(ctx context.Context, id string) ([]entity.Book, error)
	}

	BooksRepository interface {
		AddBook(ctx context.Context, book entity.Book) (entity.Book, error)
		UpdateBook(ctx context.Context, id string, name string, authorIDs []string) (entity.Book, error)
		GetBookInfo(ctx context.Context, id string) (entity.Book, error)
	}

	Transactor interface {
		WithTx(context.Context, func(ctx context.Context) error) error
	}

	OutboxRepository interface {
		SendMessage(ctx context.Context, idempotencyKey string, kind OutboxKind, message []byte) error
		GetMessages(ctx context.Context, batchSize int, inProgressTTL time.Duration) ([]OutboxData, error)
		MarkAsProcessed(ctx context.Context, idempotencyKeys []string) error
	}

	OutboxData struct {
		IdempotencyKey string
		Kind           OutboxKind
		RawData        []byte
	}
)

type OutboxKind int

const (
	OutboxKindUndefined OutboxKind = iota
	OutboxKindBook
	OutboxKindAuthor
)

func (o OutboxKind) String() string {
	switch o {
	case OutboxKindBook:
		return "book"
	case OutboxKindAuthor:
		return "author"
	default:
		return "undefined"
	}
}
