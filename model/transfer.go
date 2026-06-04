package model

const (
	TransferStatusSuccess = "success"
	TransferStatusFailed  = "failed"
)

type TransferRequest struct {
	FromID         string `json:"from_id" example:"A1"`
	ToID           string `json:"to_id" example:"A2"`
	Amount         int64  `json:"amount" example:"100"`
	IdempotencyKey string `json:"idempotency_key" example:"k7Fp2mQx9ZnW4bL8"`
}

type TransferResponse struct {
	FromID         string `json:"from_id" example:"A1"`
	ToID           string `json:"to_id" example:"A2"`
	Amount         int64  `json:"amount" example:"100"`
	IdempotencyKey string `json:"idempotency_key" example:"k7Fp2mQx9ZnW4bL8"`
}

type BatchDetail struct {
	IdempotencyKey string `json:"idempotency_key" example:"p3Rt8yKm2QnX5vJ9"`
	Status         string `json:"status" example:"success" enums:"success,failed"`
	Message        string `json:"message" example:"transfer OK"`
}

type BatchTransferResponse struct {
	Success int           `json:"success" example:"2"`
	Failed  int           `json:"failed" example:"1"`
	Details []BatchDetail `json:"details"`
}
