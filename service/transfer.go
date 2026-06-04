package service

import (
	"context"
	"errors"

	"github.com/zevinza/cita-kita-transfer-service/internal/logging"
	"github.com/zevinza/cita-kita-transfer-service/model"
	"github.com/zevinza/cita-kita-transfer-service/repository"
)

type TransferService interface {
	Transfer(ctx context.Context, request *model.TransferRequest) (*model.TransferResponse, error)
	BatchTransfer(ctx context.Context, request []model.TransferRequest) (*model.BatchTransferResponse, error)
}

type transferService struct {
	logger                logging.Logger
	transactionRepository repository.TransactionRepository
}

func NewTransferService(
	logger logging.Logger,
	transactionRepository repository.TransactionRepository,
) TransferService {
	return &transferService{
		logger:                logger,
		transactionRepository: transactionRepository,
	}
}

func (s *transferService) Transfer(ctx context.Context, request *model.TransferRequest) (*model.TransferResponse, error) {
	if _, err := s.transactionRepository.GetBalance(ctx, request.ToID); err != nil {
		return nil, err
	}

	balance, err := s.transactionRepository.GetBalance(ctx, request.FromID)
	if err != nil {
		return nil, err
	}
	if balance < request.Amount {
		s.logger.Error(ctx, "insufficient balance", "balance", balance, "amount", request.Amount)
		return nil, errors.New("insufficient balance")
	}

	resp, ok, err := s.transactionRepository.GetIdempotencyResult(ctx, request.IdempotencyKey)
	if err != nil {
		s.logger.Error(ctx, "failed to get idempotency result", "error", err)
		return nil, err
	}
	if ok {
		return resp, nil
	}

	if err := s.transactionRepository.ApplyTransfer(ctx, request); err != nil {
		s.logger.Error(ctx, "failed to apply transfer", "error", err)
		return nil, err
	}

	return &model.TransferResponse{
		FromID:         request.FromID,
		ToID:           request.ToID,
		Amount:         request.Amount,
		IdempotencyKey: request.IdempotencyKey,
	}, nil
}
