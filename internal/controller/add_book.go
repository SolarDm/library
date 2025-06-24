package controller

import (
	"context"

	"github.com/project/library/generated/api/library"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (i *implementation) AddBook(ctx context.Context, request *library.AddBookRequest) (*library.AddBookResponse, error) {
	i.logger.Info("Validating add book request")

	if err := request.ValidateAll(); err != nil {
		i.logger.Error("Error during validating add book request.", zap.Error(err))
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	book, err := i.booksUseCase.AddBook(ctx, request)

	if err != nil {
		i.logger.Error("Error during add book request.", zap.Error(err))
		return nil, err
	}

	i.logger.Info("Add book request has passed successfully")

	return book, nil
}
