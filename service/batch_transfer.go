package service

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/zevinza/cita-kita-transfer-service/model"
)

func (s *transferService) BatchTransfer(ctx context.Context, request []model.TransferRequest) (*model.BatchTransferResponse, error) {
	s.logger.Info(ctx, "batch transfer started", "count", len(request))

	wg := sync.WaitGroup{}
	results := make([]model.BatchDetail, len(request))

	for i, req := range request {
		wg.Add(1)
		go func(i int, req model.TransferRequest) {
			defer wg.Done()

			if req.IdempotencyKey == "" {
				req.IdempotencyKey = fmt.Sprintf("batch-%d-%s", i, uuid.NewString())
			}

			_, err := s.Transfer(ctx, &req)
			result := model.BatchDetail{
				IdempotencyKey: req.IdempotencyKey,
			}
			if err != nil {
				result.Message = err.Error()
				result.Status = model.TransferStatusFailed
			} else {
				result.Message = "transfer OK"
				result.Status = model.TransferStatusSuccess
			}
			results[i] = result
		}(i, req)
	}
	wg.Wait()

	var successCount, failedCount int
	for _, detail := range results {
		if detail.Status == model.TransferStatusSuccess {
			successCount++
		} else {
			failedCount++
		}
	}

	s.logger.Info(ctx, "batch transfer completed",
		"count", len(request),
		"success", successCount,
		"failed", failedCount,
	)

	return &model.BatchTransferResponse{
		Success: successCount,
		Failed:  failedCount,
		Details: results,
	}, nil
}
