package controller

import (
	"context"

	"github.com/project/library/generated/api/library"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (i *implementation) UpdateBook(ctx context.Context, request *library.UpdateBookRequest) (*library.UpdateBookResponse, error) {
	i.logger.Info("Validating update book request.")

	if err := request.ValidateAll(); err != nil {
		i.logger.Error("Error during validating update book request.", zap.Error(err))
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	resp, err := i.booksUseCase.UpdateBook(ctx, request)

	if err != nil {
		i.logger.Error("Error during update book request.", zap.Error(err))
		return nil, err
	}

	i.logger.Info("Update book request has passed successfully.")

	return resp, nil
}
