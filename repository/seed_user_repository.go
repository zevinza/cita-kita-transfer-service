package repository

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/redis/go-redis/v9"
	"github.com/zevinza/cita-kita-transfer-service/internal/logging"
	"github.com/zevinza/cita-kita-transfer-service/model"
)

//go:generate go run go.uber.org/mock/mockgen@latest -source=seed_user_repository.go -destination=seed_user_repository_mock.go -package=repository SeedUserRepository
type SeedUserRepository interface {
	SeedUserBalances(ctx context.Context) error
	ListUsers(ctx context.Context) ([]model.User, error)
}

type seedUserRepository struct {
	logger logging.Logger
	rdb    *redis.Client
}

func NewSeedUserRepository(logger logging.Logger, rdb *redis.Client) SeedUserRepository {
	return &seedUserRepository{logger: logger, rdb: rdb}
}

func (r *seedUserRepository) ListUsers(ctx context.Context) ([]model.User, error) {
	seedUsers := model.SeedUsers()
	if len(seedUsers) == 0 {
		return []model.User{}, nil
	}

	pipe := r.rdb.Pipeline()
	cmds := make([]*redis.StringCmd, len(seedUsers))
	for i, user := range seedUsers {
		key := fmt.Sprintf(BalanceKeyPrefix, user.ID)
		cmds[i] = pipe.Get(ctx, key)
	}

	if _, err := pipe.Exec(ctx); err != nil && !errors.Is(err, redis.Nil) {
		r.logger.Error(ctx, "failed to list user balances", "error", err)
		return nil, fmt.Errorf("list user balances: %w", err)
	}

	users := make([]model.User, len(seedUsers))
	for i, seed := range seedUsers {
		val, err := cmds[i].Result()
		balance := int64(0)
		if err == nil {
			balance, err = strconv.ParseInt(val, 10, 64)
			if err != nil {
				r.logger.Error(ctx, "failed to parse balance", "user_id", seed.ID, "error", err)
				return nil, fmt.Errorf("parse balance for %s: %w", seed.ID, err)
			}
		} else if !errors.Is(err, redis.Nil) {
			r.logger.Error(ctx, "failed to get balance", "user_id", seed.ID, "error", err)
			return nil, fmt.Errorf("get balance for %s: %w", seed.ID, err)
		}

		users[i] = model.User{
			ID:      seed.ID,
			Name:    seed.Name,
			Balance: balance,
		}
	}

	return users, nil
}

func (r *seedUserRepository) SeedUserBalances(ctx context.Context) error {
	users := model.SeedUsers()
	if len(users) == 0 {
		return nil
	}

	pipe := r.rdb.Pipeline()
	for _, user := range users {
		key := fmt.Sprintf(BalanceKeyPrefix, user.ID)
		pipe.SetNX(ctx, key, strconv.FormatInt(user.Balance, 10), 0)
	}

	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("seed user balances: %w", err)
	}

	return nil
}
