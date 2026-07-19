package service_test

import (
	"context"
	"testing"

	"github.com/eternal-orbit-labs/gateway/internal/domain"
	"github.com/eternal-orbit-labs/gateway/internal/service"
	"github.com/eternal-orbit-labs/gateway/internal/testutil"
)

func TestApp_List(t *testing.T) {
	repo := &testutil.FakeAppRepo{
		Apps: []*domain.App{
			{ID: "1", Slug: "alpha", Name: "Alpha App"},
			{ID: "2", Slug: "beta", Name: "Beta App"},
		},
	}
	svc := service.NewAppService(repo)

	apps, nextCursor, err := svc.List(context.Background(), "", "", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(apps) != 2 {
		t.Fatalf("expected 2 apps, got %d", len(apps))
	}
	_ = nextCursor
}

func TestApp_GetBySlug_Found(t *testing.T) {
	repo := &testutil.FakeAppRepo{
		Apps: []*domain.App{
			{ID: "1", Slug: "my-app", Name: "My App"},
		},
	}
	svc := service.NewAppService(repo)

	app, err := svc.GetBySlug(context.Background(), "my-app")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if app == nil {
		t.Fatal("expected app, got nil")
	}
	if app.Name != "My App" {
		t.Fatalf("expected 'My App', got %q", app.Name)
	}
}

func TestApp_GetBySlug_NotFound(t *testing.T) {
	repo := &testutil.FakeAppRepo{Apps: []*domain.App{}}
	svc := service.NewAppService(repo)

	app, err := svc.GetBySlug(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if app != nil {
		t.Fatal("expected nil for nonexistent slug")
	}
}
