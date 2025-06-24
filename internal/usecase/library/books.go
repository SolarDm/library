package library

import (
	"context"
	"encoding/json"

	"github.com/project/library/generated/api/library"
	"github.com/project/library/internal/usecase/repository"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/project/library/internal/entity"
)

func (l *libraryImpl) AddBook(ctx context.Context, request *library.AddBookRequest) (*library.AddBookResponse, error) {
	var book entity.Book

	err := l.transactor.WithTx(ctx, func(ctx context.Context) error {
		l.logger.Info("Add book request is being made to the database.")

		var txErr error
		book, txErr = l.booksRepository.AddBook(ctx, entity.Book{
			Name:      request.GetName(),
			AuthorIDs: request.GetAuthorIds(),
		})

		if txErr != nil {
			return txErr
		}

		serialized, txErr := json.Marshal(book)

		if txErr != nil {
			return txErr
		}

		idempotencyKey := repository.OutboxKindBook.String() + "_" + book.ID
		txErr = l.outboxRepository.SendMessage(ctx, idempotencyKey, repository.OutboxKindBook, serialized)

		if txErr != nil {
			return txErr
		}

		return nil
	})

	if err != nil {
		return nil, l.convertErr(err)
	}

	return &library.AddBookResponse{
		Book: &library.Book{
			Id:        book.ID,
			Name:      book.Name,
			AuthorIds: book.AuthorIDs,
			CreatedAt: timestamppb.New(book.CreatedAt),
			UpdatedAt: timestamppb.New(book.UpdatedAt),
		},
	}, nil
}

func (l *libraryImpl) UpdateBook(ctx context.Context, request *library.UpdateBookRequest) (*library.UpdateBookResponse, error) {
	l.logger.Info("Update book request is being made to the database.")
	_, err := l.booksRepository.UpdateBook(ctx, request.GetId(), request.GetName(), request.GetAuthorIds())

	if err != nil {
		return nil, l.convertErr(err)
	}

	return &library.UpdateBookResponse{}, nil
}

func (l *libraryImpl) GetBookInfo(ctx context.Context, request *library.GetBookInfoRequest) (*library.GetBookInfoResponse, error) {
	l.logger.Info("Get book info request is being made to the database.")
	book, err := l.booksRepository.GetBookInfo(ctx, request.GetId())

	if err != nil {
		return nil, l.convertErr(err)
	}

	return &library.GetBookInfoResponse{
		Book: &library.Book{
			Id:        book.ID,
			Name:      book.Name,
			AuthorIds: book.AuthorIDs,
			CreatedAt: timestamppb.New(book.CreatedAt),
			UpdatedAt: timestamppb.New(book.UpdatedAt),
		},
	}, nil
}
