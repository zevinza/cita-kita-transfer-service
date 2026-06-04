package service

import (
	"context"

	"github.com/zevinza/cita-kita-transfer-service/internal/logging"
	"github.com/zevinza/cita-kita-transfer-service/model"
	"github.com/zevinza/cita-kita-transfer-service/repository"
)

type TransferService interface {
	Transfer(ctx context.Context, request model.TransferRequest) (model.TransferResponse, error)
	BatchTransfer(ctx context.Context, request []model.TransferRequest) (model.BatchTransferResponse, error)
}

type transferService struct {
	logger                logging.Logger
	transactionRepository repository.TransactionRepository
}

func NewTransferService(logger logging.Logger, transactionRepository repository.TransactionRepository) TransferService {
	return &transferService{logger: logger, transactionRepository: transactionRepository}
}

func (s *transferService) Transfer(ctx context.Context, request model.TransferRequest) (model.TransferResponse, error) {
	return model.TransferResponse{}, nil
}
