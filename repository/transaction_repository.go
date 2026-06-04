package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/redis/go-redis/v9"
	"github.com/zevinza/cita-kita-transfer-service/internal/logging"
	"github.com/zevinza/cita-kita-transfer-service/model"
)

const (
	BalanceKeyPrefix     = "balance:%s"
	IdempotencyKeyPrefix = "idempotency:%s"
)

// applyTransferScript: atomic idempotency check + debit/credit. status 0=idem exists, 1=applied, -1=insufficient.
const applyTransferScript = `
local idem_key = KEYS[1]
local from_key = KEYS[2]
local to_key = KEYS[3]
local amount = tonumber(ARGV[1])
local idem_value = ARGV[2]

local existing = redis.call('GET', idem_key)
if existing then
    return {0, existing}
end

local from_balance = redis.call('GET', from_key)
if not from_balance then
    return {-1, 'insufficient balance'}
end
from_balance = tonumber(from_balance)
if from_balance < amount then
    return {-1, 'insufficient balance'}
end

local to_balance = redis.call('GET', to_key)
if not to_balance then
    to_balance = 0
else
    to_balance = tonumber(to_balance)
end

redis.call('SET', from_key, from_balance - amount)
redis.call('SET', to_key, to_balance + amount)
redis.call('SET', idem_key, idem_value)

return {1, idem_value}
`

//go:generate go run go.uber.org/mock/mockgen@latest -source=transaction_repository.go -destination=transaction_repository_mock.go -package=repository TransactionRepository
type TransactionRepository interface {
	GetBalance(ctx context.Context, accountID string) (int64, error)
	GetIdempotencyResult(ctx context.Context, key string) (*model.TransferResponse, bool, error)
	ApplyTransfer(ctx context.Context, req *model.TransferRequest) error
}

type transactionRepository struct {
	logger logging.Logger
	rdb    *redis.Client
}

func NewTransactionRepository(logger logging.Logger, rdb *redis.Client) TransactionRepository {
	return &transactionRepository{logger: logger, rdb: rdb}
}

func (r *transactionRepository) GetBalance(ctx context.Context, accountID string) (int64, error) {
	key := fmt.Sprintf(BalanceKeyPrefix, accountID)
	val, err := r.rdb.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			r.logger.Error(ctx, "account not found", "account_id", accountID)
			return 0, fmt.Errorf("account not found")
		}
		r.logger.Error(ctx, "failed to get balance", "error", err)
		return 0, fmt.Errorf("failed to get balance: %w", err)
	}
	return strconv.ParseInt(val, 10, 64)
}

func (r *transactionRepository) GetIdempotencyResult(ctx context.Context, key string) (*model.TransferResponse, bool, error) {
	idemKey := fmt.Sprintf(IdempotencyKeyPrefix, key)
	val, err := r.rdb.Get(ctx, idemKey).Result()
	if errors.Is(err, redis.Nil) {
		return nil, false, nil
	}
	if err != nil {
		r.logger.Error(ctx, "failed to get idempotency result", "error", err)
		return nil, false, err
	}

	var result model.TransferResponse
	if err := json.Unmarshal([]byte(val), &result); err != nil {
		r.logger.Error(ctx, "failed to unmarshal idempotency result", "error", err)
		return nil, false, err
	}

	return &result, true, nil
}

func (r *transactionRepository) ApplyTransfer(ctx context.Context, req *model.TransferRequest) error {
	resp := &model.TransferResponse{
		FromID:         req.FromID,
		ToID:           req.ToID,
		Amount:         req.Amount,
		IdempotencyKey: req.IdempotencyKey,
	}
	idemJSON, err := json.Marshal(resp)
	if err != nil {
		r.logger.Error(ctx, "failed to marshal idempotency result", "error", err)
		return fmt.Errorf("failed to marshal idempotency result: %w", err)
	}

	keys := []string{
		fmt.Sprintf(IdempotencyKeyPrefix, req.IdempotencyKey),
		fmt.Sprintf(BalanceKeyPrefix, req.FromID),
		fmt.Sprintf(BalanceKeyPrefix, req.ToID),
	}

	result, err := r.rdb.Eval(ctx, applyTransferScript, keys, req.Amount, string(idemJSON)).Result()
	if err != nil {
		r.logger.Error(ctx, "failed to evaluate apply transfer script", "error", err)
		return fmt.Errorf("failed to evaluate apply transfer script: %w", err)
	}

	values, ok := result.([]any)
	if !ok || len(values) != 2 {
		r.logger.Error(ctx, "unexpected apply transfer result", "result", result)
		return fmt.Errorf("unexpected apply transfer result: %v", result)
	}

	status, err := strconv.ParseInt(fmt.Sprint(values[0]), 10, 64)
	if err != nil {
		r.logger.Error(ctx, "failed to parse apply transfer status", "error", err)
		return fmt.Errorf("parse apply transfer status: %w", err)
	}

	switch status {
	case 0, 1:
		return nil
	case -1:
		r.logger.Error(ctx, "insufficient balance", "from_id", req.FromID, "to_id", req.ToID, "amount", req.Amount)
		return fmt.Errorf("insufficient balance")
	default:
		r.logger.Error(ctx, "unknown apply transfer status", "status", status)
		return fmt.Errorf("unknown apply transfer status: %d", status)
	}
}
