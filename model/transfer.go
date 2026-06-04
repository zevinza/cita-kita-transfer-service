package model

type TransferRequest struct {
	FromID         string `json:"from_id"`
	ToID           string `json:"to_id"`
	Amount         int64  `json:"amount"`
	IdempotencyKey string `json:"idempotency_key"`
}

type TransferResponse struct {
	FromID         string  `json:"from_id"`
	ToID           string  `json:"to_id"`
	Amount         int64   `json:"amount"`
	IdempotencyKey string  `json:"idempotency_key"`
	Message        *string `json:"message,omitempty"`
}

type BatchTransferResponse struct {
	Success int                `json:"success"`
	Failed  int                `json:"failed"`
	Details []TransferResponse `json:"details"`
}
