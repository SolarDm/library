package controller

import (
	"context"

	"github.com/project/library/generated/api/library"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (i *implementation) RegisterAuthor(ctx context.Context, request *library.RegisterAuthorRequest) (*library.RegisterAuthorResponse, error) {
	i.logger.Info("Validating register author request.")

	if err := request.ValidateAll(); err != nil {
		i.logger.Error("Error during validating register author request.", zap.Error(err))
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	author, err := i.authorUseCase.RegisterAuthor(ctx, request)

	if err != nil {
		i.logger.Error("Error during register author request.", zap.Error(err))
		return nil, err
	}

	i.logger.Info("Register author request has passed successfully.")

	return author, nil
}
