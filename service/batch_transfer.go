package service

import (
	"context"

	"github.com/zevinza/cita-kita-transfer-service/model"
)

func (s *transferService) BatchTransfer(ctx context.Context, request []model.TransferRequest) (model.BatchTransferResponse, error) {
	return model.BatchTransferResponse{}, nil
}
