package repository

import (
	"context"

	"github.com/zevinza/cita-kita-transfer-service/internal/logging"
	"github.com/zevinza/cita-kita-transfer-service/model"
)

type TransactionRepository interface {
	GetBalance(ctx context.Context, accountID string) (int64, error)
	GetIdempotencyResult(ctx context.Context, key string) (model.TransferResponse, bool, error)
	ApplyTransfer(ctx context.Context, req model.TransferRequest) (model.TransferResponse, error)
}

type transactionRepository struct {
	logger logging.Logger
}

func NewTransactionRepository(logger logging.Logger) TransactionRepository {
	return &transactionRepository{logger: logger}
}

func (r *transactionRepository) GetBalance(ctx context.Context, accountID string) (int64, error) {
	return 0, nil
}

func (r *transactionRepository) GetIdempotencyResult(ctx context.Context, key string) (model.TransferResponse, bool, error) {
	return model.TransferResponse{}, false, nil
}

func (r *transactionRepository) ApplyTransfer(ctx context.Context, req model.TransferRequest) (model.TransferResponse, error) {
	return model.TransferResponse{}, nil
}
