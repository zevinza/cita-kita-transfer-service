package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/zevinza/cita-kita-transfer-service/internal/config"
	"github.com/zevinza/cita-kita-transfer-service/internal/lock"
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
	accountLocker         lock.AccountLocker
}

func NewTransferService(
	logger logging.Logger,
	transactionRepository repository.TransactionRepository,
	accountLocker lock.AccountLocker,
) TransferService {
	return &transferService{
		logger:                logger,
		transactionRepository: transactionRepository,
		accountLocker:         accountLocker,
	}
}

func (s *transferService) Transfer(ctx context.Context, request *model.TransferRequest) (*model.TransferResponse, error) {
	if request.FromID == request.ToID {
		return nil, errors.New("from_id and to_id cannot be the same")
	}

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

	s.accountLocker.Lock(request.FromID, request.ToID)
	defer s.accountLocker.Unlock(request.FromID, request.ToID)

	if err := s.applyTransferWithRetry(ctx, request); err != nil {
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

func (s *transferService) applyTransferWithRetry(ctx context.Context, request *model.TransferRequest) error {
	cfg := config.Get()
	var lastErr error

	for attempt := 0; attempt <= cfg.MaxRetries; attempt++ {
		lastErr = s.transactionRepository.ApplyTransfer(ctx, request)
		if lastErr == nil {
			return nil
		}
		if strings.Contains(lastErr.Error(), "insufficient balance") {
			return lastErr
		}
		if attempt == cfg.MaxRetries {
			break
		}

		timer := time.NewTimer(cfg.RetryBaseBackoff)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}

	return lastErr
}

func matchesIdempotencyRequest(resp *model.TransferResponse, req *model.TransferRequest) bool {
	return resp.FromID == req.FromID &&
		resp.ToID == req.ToID &&
		resp.Amount == req.Amount
}
