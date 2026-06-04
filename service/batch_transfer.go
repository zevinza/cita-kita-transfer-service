package service

import (
	"context"
	"sync"

	"github.com/zevinza/cita-kita-transfer-service/model"
)

func (s *transferService) BatchTransfer(ctx context.Context, request []model.TransferRequest) (*model.BatchTransferResponse, error) {
	wg := sync.WaitGroup{}
	var successCount, failedCount int
	results := make([]model.BatchDetail, len(request))

	for i, req := range request {
		wg.Add(1)
		go func(req model.TransferRequest) {
			defer wg.Done()
			resp, err := s.Transfer(ctx, &req)
			result := model.BatchDetail{
				IdempotencyKey: resp.IdempotencyKey,
			}
			if err != nil {
				failedCount++
				result.Message = err.Error()
				result.Status = model.TransferStatusFailed
			} else {
				successCount++
				result.Message = "transfer OK"
				result.Status = model.TransferStatusSuccess
			}
			results[i] = result
		}(req)
	}
	wg.Wait()
	return &model.BatchTransferResponse{
		Success: successCount,
		Failed:  failedCount,
		Details: results,
	}, nil
}
