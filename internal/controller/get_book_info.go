package controller

import (
	"context"

	"github.com/project/library/generated/api/library"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (i *implementation) GetBookInfo(ctx context.Context, request *library.GetBookInfoRequest) (*library.GetBookInfoResponse, error) {
	i.logger.Info("Validating get book info request.")

	if err := request.ValidateAll(); err != nil {
		i.logger.Error("Error during validating get book info request.", zap.Error(err))
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	book, err := i.booksUseCase.GetBookInfo(ctx, request)

	if err != nil {
		i.logger.Error("Error during get book info request.", zap.Error(err))
		return nil, err
	}

	i.logger.Info("Change author request has passed successfully.")

	return book, nil
}
