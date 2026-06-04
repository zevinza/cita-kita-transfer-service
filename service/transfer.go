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
	fields := transferLogFields(request)
	s.logger.Info(ctx, "transfer started", fields...)

	if request.FromID == request.ToID {
		s.logger.Warn(ctx, "transfer rejected: same account", fields...)
		return nil, errors.New("from_id and to_id cannot be the same")
	}

	if _, err := s.transactionRepository.GetBalance(ctx, request.ToID); err != nil {
		s.logger.Error(ctx, "transfer failed: destination lookup", append(fields, "error", err)...)
		return nil, err
	}

	balance, err := s.transactionRepository.GetBalance(ctx, request.FromID)
	if err != nil {
		s.logger.Error(ctx, "transfer failed: source lookup", append(fields, "error", err)...)
		return nil, err
	}
	if balance < request.Amount {
		s.logger.Error(ctx, "transfer rejected: insufficient balance", append(fields, "balance", balance)...)
		return nil, errors.New("insufficient balance")
	}

	resp, ok, err := s.transactionRepository.GetIdempotencyResult(ctx, request.IdempotencyKey)
	if err != nil {
		s.logger.Error(ctx, "transfer failed: idempotency lookup", append(fields, "error", err)...)
		return nil, err
	}
	if ok {
		s.logger.Warn(ctx, "transfer idempotent replay", fields...)
		return resp, nil
	}

	s.accountLocker.Lock(request.FromID, request.ToID)
	defer s.accountLocker.Unlock(request.FromID, request.ToID)

	if err := s.applyTransferWithRetry(ctx, request); err != nil {
		s.logger.Error(ctx, "transfer failed: apply", append(fields, "error", err)...)
		return nil, err
	}

	s.logger.Info(ctx, "transfer applied", fields...)
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

	fields := transferLogFields(request)

	for attempt := 0; attempt <= cfg.MaxRetries; attempt++ {
		lastErr = s.transactionRepository.ApplyTransfer(ctx, request)
		if lastErr == nil {
			return nil
		}
		if strings.Contains(lastErr.Error(), "insufficient balance") {
			s.logger.Warn(ctx, "transfer apply rejected: insufficient balance", append(fields, "attempt", attempt)...)
			return lastErr
		}
		if attempt == cfg.MaxRetries {
			break
		}

		s.logger.Warn(ctx, "transfer apply retry", append(fields, "attempt", attempt, "error", lastErr)...)

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

func transferLogFields(request *model.TransferRequest) []any {
	return []any{
		"idempotency_key", request.IdempotencyKey,
		"from_id", request.FromID,
		"to_id", request.ToID,
		"amount", request.Amount,
	}
}
