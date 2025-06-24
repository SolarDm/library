package controller

import (
	"github.com/project/library/generated/api/library"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (i *implementation) GetAuthorBooks(request *library.GetAuthorBooksRequest, server library.Library_GetAuthorBooksServer) error {
	i.logger.Info("Validating get author books request.")

	if err := request.ValidateAll(); err != nil {
		i.logger.Error("Error during validating get author books request.", zap.Error(err))
		return status.Error(codes.InvalidArgument, err.Error())
	}

	err := i.authorUseCase.GetAuthorBooks(server.Context(), request, server)

	if err != nil {
		i.logger.Error("Error during get author books request.", zap.Error(err))
		return err
	}

	i.logger.Info("Add book request has passed successfully.")

	return nil
}
