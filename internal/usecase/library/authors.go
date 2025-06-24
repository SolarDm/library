package library

import (
	"context"
	"encoding/json"

	"github.com/project/library/generated/api/library"
	"github.com/project/library/internal/usecase/repository"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/project/library/internal/entity"
)

func (l *libraryImpl) RegisterAuthor(ctx context.Context, request *library.RegisterAuthorRequest) (*library.RegisterAuthorResponse, error) {
	var author entity.Author

	err := l.transactor.WithTx(ctx, func(ctx context.Context) error {
		l.logger.Info("Register author info request is being made to the database.")

		var txErr error
		author, txErr = l.authorRepository.RegisterAuthor(ctx, entity.Author{
			Name: request.GetName(),
		})

		if txErr != nil {
			return txErr
		}

		serialized, txErr := json.Marshal(author)

		if txErr != nil {
			return txErr
		}

		idempotencyKey := repository.OutboxKindAuthor.String() + "_" + author.ID
		txErr = l.outboxRepository.SendMessage(ctx, idempotencyKey, repository.OutboxKindAuthor, serialized)

		if txErr != nil {
			return txErr
		}

		return nil
	})

	if err != nil {
		return nil, l.convertErr(err)
	}

	return &library.RegisterAuthorResponse{Id: author.ID}, nil
}

func (l *libraryImpl) ChangeAuthorInfo(ctx context.Context, request *library.ChangeAuthorInfoRequest) (*library.ChangeAuthorInfoResponse, error) {
	l.logger.Info("Change author info request is being made to the database.")
	_, err := l.authorRepository.ChangeAuthorInfo(ctx, request.GetId(), request.GetName())

	if err != nil {
		return nil, l.convertErr(err)
	}

	return &library.ChangeAuthorInfoResponse{}, err
}

func (l *libraryImpl) GetAuthorInfo(ctx context.Context, request *library.GetAuthorInfoRequest) (*library.GetAuthorInfoResponse, error) {
	l.logger.Info("Get author info request is being made to the database.")
	author, err := l.authorRepository.GetAuthorInfo(ctx, request.GetId())

	if err != nil {
		return nil, l.convertErr(err)
	}

	return &library.GetAuthorInfoResponse{
		Id:   author.ID,
		Name: author.Name,
	}, nil
}

func (l *libraryImpl) GetAuthorBooks(ctx context.Context, request *library.GetAuthorBooksRequest, resp library.Library_GetAuthorBooksServer) error {
	l.logger.Info("Get author books request is being made to the database.")
	books, err := l.authorRepository.GetAuthorBooks(ctx, request.GetAuthorId())

	if err != nil {
		return l.convertErr(err)
	}

	for _, book := range books {
		err = resp.Send(&library.Book{
			Id:        book.ID,
			Name:      book.Name,
			AuthorIds: book.AuthorIDs,
			CreatedAt: timestamppb.New(book.CreatedAt),
			UpdatedAt: timestamppb.New(book.UpdatedAt),
		})
		if err != nil {
			l.logger.Error("error while sending response", zap.Error(err))
		}
	}

	return nil
}
