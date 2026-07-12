package service

import (
	"context"

	"github.com/eternal-orbit-labs/gateway/internal/domain"
)

type AppService struct {
	apps domain.AppRepository
}

func NewAppService(apps domain.AppRepository) *AppService {
	return &AppService{apps: apps}
}

func (s *AppService) List(ctx context.Context, q, cursor string, limit int) ([]*domain.App, string, error) {
	return s.apps.List(ctx, q, cursor, limit)
}

func (s *AppService) GetBySlug(ctx context.Context, slug string) (*domain.App, error) {
	return s.apps.GetBySlug(ctx, slug)
}
