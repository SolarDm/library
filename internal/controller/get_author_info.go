package controller

import (
	"context"

	"github.com/project/library/generated/api/library"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (i *implementation) GetAuthorInfo(ctx context.Context, request *library.GetAuthorInfoRequest) (*library.GetAuthorInfoResponse, error) {
	i.logger.Info("Validating get author info request.")

	if err := request.ValidateAll(); err != nil {
		i.logger.Error("Error during validating get author info request.", zap.Error(err))
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	author, err := i.authorUseCase.GetAuthorInfo(ctx, request)

	if err != nil {
		i.logger.Error("Error during get author info request.", zap.Error(err))
		return nil, err
	}

	i.logger.Info("Get author info request has passed successfully.")

	return author, nil
}
