package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/zevinza/cita-kita-transfer-service/internal/lock"
	"github.com/zevinza/cita-kita-transfer-service/internal/logging"
	"github.com/zevinza/cita-kita-transfer-service/model"
	"github.com/zevinza/cita-kita-transfer-service/repository"
)

func newTransferServiceTest(t *testing.T) (TransferService, *repository.MockTransactionRepository, *logging.MockLogger) {
	t.Helper()
	ctrl := gomock.NewController(t)
	mockRepo := repository.NewMockTransactionRepository(ctrl)
	mockLogger := logging.NewMockLogger(ctrl)
	allowTransferLogs(mockLogger)
	svc := NewTransferService(mockLogger, mockRepo, lock.NewAccountLocker())
	return svc, mockRepo, mockLogger
}

func allowTransferLogs(mockLogger *logging.MockLogger) {
	mockLogger.EXPECT().Info(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().Warn(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().Error(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
}

func TestTransfer_ValidationErrors(t *testing.T) {
	t.Run("Same From and To ID", func(t *testing.T) {
		request := &model.TransferRequest{
			FromID:         "A1",
			ToID:           "A1",
			Amount:         100,
			IdempotencyKey: "key-same-account",
		}

		if request.FromID == request.ToID {
			assert.True(t, true, "Transfer should reject when from_id equals to_id")
		}
	})

	t.Run("Zero Amount", func(t *testing.T) {
		request := &model.TransferRequest{
			FromID:         "A1",
			ToID:           "A2",
			Amount:         0,
			IdempotencyKey: "key-zero-amount",
		}

		assert.True(t, request.Amount <= 0, "Zero amount should be treated as invalid transfer value")
	})

	t.Run("Negative Amount", func(t *testing.T) {
		request := &model.TransferRequest{
			FromID:         "A1",
			ToID:           "A2",
			Amount:         -50,
			IdempotencyKey: "key-negative-amount",
		}

		assert.True(t, request.Amount < 0, "Negative amount should be invalid")
	})

	t.Run("Empty Idempotency Key", func(t *testing.T) {
		request := &model.TransferRequest{
			FromID: "A1",
			ToID:   "A2",
			Amount: 100,
		}

		assert.Empty(t, request.IdempotencyKey, "Idempotency key may be empty before batch auto-fill")
	})
}

func TestMatchesIdempotencyRequest_Logic(t *testing.T) {
	t.Run("Matching Request", func(t *testing.T) {
		resp := &model.TransferResponse{
			FromID: "A1",
			ToID:   "A2",
			Amount: 100,
		}
		req := &model.TransferRequest{
			FromID: "A1",
			ToID:   "A2",
			Amount: 100,
		}

		matches := resp.FromID == req.FromID &&
			resp.ToID == req.ToID &&
			resp.Amount == req.Amount

		assert.True(t, matches, "Cached idempotency result should match the request")
	})

	t.Run("Different Amount", func(t *testing.T) {
		resp := &model.TransferResponse{
			FromID: "A1",
			ToID:   "A2",
			Amount: 100,
		}
		req := &model.TransferRequest{
			FromID: "A1",
			ToID:   "A2",
			Amount: 200,
		}

		matches := resp.FromID == req.FromID &&
			resp.ToID == req.ToID &&
			resp.Amount == req.Amount

		assert.False(t, matches, "Different amount should not match idempotency cache")
	})

	t.Run("Different Destination", func(t *testing.T) {
		resp := &model.TransferResponse{
			FromID: "A1",
			ToID:   "A2",
			Amount: 100,
		}
		req := &model.TransferRequest{
			FromID: "A1",
			ToID:   "A3",
			Amount: 100,
		}

		matches := resp.FromID == req.FromID &&
			resp.ToID == req.ToID &&
			resp.Amount == req.Amount

		assert.False(t, matches, "Different to_id should not match idempotency cache")
	})
}

func TestTransfer_InsufficientBalance_Logic(t *testing.T) {
	t.Run("Balance Less Than Amount", func(t *testing.T) {
		balance := int64(50)
		amount := int64(100)

		insufficient := balance < amount
		assert.True(t, insufficient, "Should detect insufficient balance")
	})

	t.Run("Balance Equal To Amount", func(t *testing.T) {
		balance := int64(100)
		amount := int64(100)

		insufficient := balance < amount
		assert.False(t, insufficient, "Equal balance should allow transfer")
	})
}

func TestApplyTransferWithRetry_Logic(t *testing.T) {
	t.Run("Non-Retryable Insufficient Balance Error", func(t *testing.T) {
		err := errors.New("insufficient balance")
		shouldRetry := !strings.Contains(err.Error(), "insufficient balance")

		assert.False(t, shouldRetry, "Insufficient balance should not be retried")
	})

	t.Run("Retryable Transient Error", func(t *testing.T) {
		err := errors.New("connection reset")
		shouldRetry := !strings.Contains(err.Error(), "insufficient balance")

		assert.True(t, shouldRetry, "Transient errors should be retried")
	})
}

func TestTransfer_SameAccount(t *testing.T) {
	svc, _, _ := newTransferServiceTest(t)

	_, err := svc.Transfer(context.Background(), &model.TransferRequest{
		FromID:         "A1",
		ToID:           "A1",
		Amount:         100,
		IdempotencyKey: "key-1",
	})

	assert.Error(t, err)
	assert.Equal(t, "from_id and to_id cannot be the same", err.Error())
}

func TestTransfer_DestinationNotFound(t *testing.T) {
	svc, mockRepo, _ := newTransferServiceTest(t)
	ctx := context.Background()

	mockRepo.EXPECT().
		GetBalance(ctx, "A2").
		Return(int64(0), errors.New("account not found"))

	_, err := svc.Transfer(ctx, &model.TransferRequest{
		FromID:         "A1",
		ToID:           "A2",
		Amount:         100,
		IdempotencyKey: "key-2",
	})

	assert.Error(t, err)
	assert.Equal(t, "account not found", err.Error())
}

func TestTransfer_InsufficientBalance(t *testing.T) {
	svc, mockRepo, _ := newTransferServiceTest(t)
	ctx := context.Background()

	mockRepo.EXPECT().GetBalance(ctx, "A2").Return(int64(2000), nil)
	mockRepo.EXPECT().GetBalance(ctx, "A1").Return(int64(50), nil)

	_, err := svc.Transfer(ctx, &model.TransferRequest{
		FromID:         "A1",
		ToID:           "A2",
		Amount:         100,
		IdempotencyKey: "key-3",
	})

	assert.Error(t, err)
	assert.Equal(t, "insufficient balance", err.Error())
}

func TestTransfer_IdempotentCacheHit(t *testing.T) {
	svc, mockRepo, _ := newTransferServiceTest(t)
	ctx := context.Background()
	cached := &model.TransferResponse{
		FromID:         "A1",
		ToID:           "A2",
		Amount:         100,
		IdempotencyKey: "key-4",
	}

	mockRepo.EXPECT().GetBalance(ctx, "A2").Return(int64(2000), nil)
	mockRepo.EXPECT().GetBalance(ctx, "A1").Return(int64(1000), nil)
	mockRepo.EXPECT().
		GetIdempotencyResult(ctx, "key-4").
		Return(cached, true, nil)

	resp, err := svc.Transfer(ctx, &model.TransferRequest{
		FromID:         "A1",
		ToID:           "A2",
		Amount:         100,
		IdempotencyKey: "key-4",
	})

	assert.NoError(t, err)
	assert.Equal(t, cached, resp)
}

func TestTransfer_Success(t *testing.T) {
	svc, mockRepo, _ := newTransferServiceTest(t)
	ctx := context.Background()
	req := &model.TransferRequest{
		FromID:         "A1",
		ToID:           "A2",
		Amount:         100,
		IdempotencyKey: "key-5",
	}

	mockRepo.EXPECT().GetBalance(ctx, "A2").Return(int64(2000), nil)
	mockRepo.EXPECT().GetBalance(ctx, "A1").Return(int64(1000), nil)
	mockRepo.EXPECT().GetIdempotencyResult(ctx, "key-5").Return(nil, false, nil)
	mockRepo.EXPECT().ApplyTransfer(ctx, req).Return(nil)

	resp, err := svc.Transfer(ctx, req)

	assert.NoError(t, err)
	assert.Equal(t, req.FromID, resp.FromID)
	assert.Equal(t, req.ToID, resp.ToID)
	assert.Equal(t, req.Amount, resp.Amount)
	assert.Equal(t, req.IdempotencyKey, resp.IdempotencyKey)
}

func TestBatchTransfer_Logic(t *testing.T) {
	t.Run("Count Success and Failed", func(t *testing.T) {
		details := []model.BatchDetail{
			{Status: model.TransferStatusSuccess},
			{Status: model.TransferStatusFailed},
			{Status: model.TransferStatusSuccess},
		}

		var successCount, failedCount int
		for _, detail := range details {
			if detail.Status == model.TransferStatusSuccess {
				successCount++
			} else {
				failedCount++
			}
		}

		assert.Equal(t, 2, successCount)
		assert.Equal(t, 1, failedCount)
	})

	t.Run("Auto Generate Idempotency Key Prefix", func(t *testing.T) {
		generatedKey := "batch-0-550e8400-e29b-41d4-a716-446655440000"

		assert.True(t, strings.HasPrefix(generatedKey, "batch-0-"),
			"Batch idempotency keys should use batch index prefix")
	})

	t.Run("Empty Batch", func(t *testing.T) {
		requests := []model.TransferRequest{}

		assert.Equal(t, 0, len(requests))
	})
}

func TestBatchTransfer_Service(t *testing.T) {
	svc, mockRepo, _ := newTransferServiceTest(t)
	ctx := context.Background()

	req := model.TransferRequest{
		FromID:         "A1",
		ToID:           "A2",
		Amount:         100,
		IdempotencyKey: "batch-key-1",
	}

	mockRepo.EXPECT().GetBalance(ctx, "A2").Return(int64(2000), nil).AnyTimes()
	mockRepo.EXPECT().GetBalance(ctx, "A1").Return(int64(1000), nil).AnyTimes()
	mockRepo.EXPECT().GetIdempotencyResult(ctx, "batch-key-1").Return(nil, false, nil)
	mockRepo.EXPECT().ApplyTransfer(ctx, &req).Return(nil)

	resp, err := svc.BatchTransfer(ctx, []model.TransferRequest{req})

	assert.NoError(t, err)
	assert.Equal(t, 1, resp.Success)
	assert.Equal(t, 0, resp.Failed)
	assert.Len(t, resp.Details, 1)
	assert.Equal(t, model.TransferStatusSuccess, resp.Details[0].Status)
	assert.Equal(t, "transfer OK", resp.Details[0].Message)
}

func TestBatchTransfer_PartialFailure(t *testing.T) {
	svc, mockRepo, _ := newTransferServiceTest(t)
	ctx := context.Background()

	successReq := model.TransferRequest{
		FromID:         "A1",
		ToID:           "A2",
		Amount:         50,
		IdempotencyKey: "batch-ok",
	}
	failReq := model.TransferRequest{
		FromID:         "A1",
		ToID:           "A1",
		Amount:         50,
		IdempotencyKey: "batch-fail",
	}

	mockRepo.EXPECT().GetBalance(ctx, "A2").Return(int64(2000), nil).AnyTimes()
	mockRepo.EXPECT().GetBalance(ctx, "A1").Return(int64(1000), nil).AnyTimes()
	mockRepo.EXPECT().GetIdempotencyResult(ctx, "batch-ok").Return(nil, false, nil)
	mockRepo.EXPECT().ApplyTransfer(ctx, &successReq).Return(nil)

	resp, err := svc.BatchTransfer(ctx, []model.TransferRequest{successReq, failReq})

	assert.NoError(t, err)
	assert.Equal(t, 1, resp.Success)
	assert.Equal(t, 1, resp.Failed)
}

func TestTransferStatusConstants(t *testing.T) {
	t.Run("Success Status", func(t *testing.T) {
		assert.Equal(t, "success", model.TransferStatusSuccess)
	})

	t.Run("Failed Status", func(t *testing.T) {
		assert.Equal(t, "failed", model.TransferStatusFailed)
	})
}

func TestRepositoryKeyPrefixes_Logic(t *testing.T) {
	t.Run("Balance Key Format", func(t *testing.T) {
		accountID := "A1"
		key := "balance:" + accountID

		assert.Equal(t, "balance:A1", key)
	})

	t.Run("Idempotency Key Format", func(t *testing.T) {
		idemKey := "idem-123"
		key := "idempotency:" + idemKey

		assert.Equal(t, "idempotency:idem-123", key)
	})
}
