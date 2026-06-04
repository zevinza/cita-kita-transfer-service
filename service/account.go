package service

import (
	"context"

	"github.com/zevinza/cita-kita-transfer-service/internal/logging"
	"github.com/zevinza/cita-kita-transfer-service/model"
	"github.com/zevinza/cita-kita-transfer-service/repository"
)

type AccountService interface {
	ListAccounts(ctx context.Context) (*model.AccountListResponse, error)
}

type accountService struct {
	logger             logging.Logger
	seedUserRepository repository.SeedUserRepository
}

func NewAccountService(
	logger logging.Logger,
	seedUserRepository repository.SeedUserRepository,
) AccountService {
	return &accountService{
		logger:             logger,
		seedUserRepository: seedUserRepository,
	}
}

func (s *accountService) ListAccounts(ctx context.Context) (*model.AccountListResponse, error) {
	users, err := s.seedUserRepository.ListUsers(ctx)
	if err != nil {
		s.logger.Error(ctx, "failed to list accounts", "error", err)
		return nil, err
	}

	return &model.AccountListResponse{Users: users}, nil
}
