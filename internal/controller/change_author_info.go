package controller

import (
	"context"

	"github.com/project/library/generated/api/library"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (i *implementation) ChangeAuthorInfo(ctx context.Context, request *library.ChangeAuthorInfoRequest) (*library.ChangeAuthorInfoResponse, error) {
	i.logger.Info("Validating change author request.")

	if err := request.ValidateAll(); err != nil {
		i.logger.Error("Error during validating change author info request.", zap.Error(err))
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	resp, err := i.authorUseCase.ChangeAuthorInfo(ctx, request)

	if err != nil {
		i.logger.Error("Error during change author info request.", zap.Error(err))
		return nil, err
	}

	i.logger.Info("Change author request has passed successfully.")

	return resp, nil
}
