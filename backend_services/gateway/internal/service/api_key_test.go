package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/eternal-orbit-labs/gateway/internal/service"
	"github.com/eternal-orbit-labs/gateway/internal/testutil"
)

func TestAPIKey_Create(t *testing.T) {
	svc, _ := testutil.NewTestAPIKeyService()
	result, err := svc.Create(context.Background(), "user-1", "My Key")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Key == nil {
		t.Fatal("expected key")
	}
	if result.RawValue == "" {
		t.Fatal("expected raw value")
	}
	if result.Key.Label != "My Key" {
		t.Fatalf("expected label 'My Key', got %q", result.Key.Label)
	}
	if result.Key.UserID != "user-1" {
		t.Fatalf("expected userID 'user-1', got %q", result.Key.UserID)
	}
}

func TestAPIKey_List(t *testing.T) {
	svc, _ := testutil.NewTestAPIKeyService()
	ctx := context.Background()
	svc.Create(ctx, "user-1", "Key A")
	svc.Create(ctx, "user-1", "Key B")
	svc.Create(ctx, "user-2", "Key C")

	keys, err := svc.List(ctx, "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(keys) != 2 {
		t.Fatalf("expected 2 keys, got %d", len(keys))
	}
}

func TestAPIKey_Delete(t *testing.T) {
	svc, _ := testutil.NewTestAPIKeyService()
	ctx := context.Background()
	result, _ := svc.Create(ctx, "user-1", "To Delete")
	err := svc.Delete(ctx, result.Key.ID, "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	keys, _ := svc.List(ctx, "user-1")
	if len(keys) != 0 {
		t.Fatalf("expected 0 keys after delete, got %d", len(keys))
	}
}

func TestAPIKey_Authenticate_Success(t *testing.T) {
	svc, _ := testutil.NewTestAPIKeyService()
	ctx := context.Background()
	result, _ := svc.Create(ctx, "user-1", "Auth Key")

	key, err := svc.Authenticate(ctx, result.RawValue)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if key.UserID != "user-1" {
		t.Fatalf("expected userID 'user-1', got %q", key.UserID)
	}
}

func TestAPIKey_Authenticate_InvalidFormat(t *testing.T) {
	svc, _ := testutil.NewTestAPIKeyService()
	_, err := svc.Authenticate(context.Background(), "not-a-valid-key")
	if !errors.Is(err, service.ErrInvalidAPIKeyFormat) {
		t.Fatalf("expected ErrInvalidAPIKeyFormat, got %v", err)
	}
}

func TestAPIKey_Authenticate_NonexistentKey(t *testing.T) {
	svc, _ := testutil.NewTestAPIKeyService()
	_, err := svc.Authenticate(context.Background(), "eol_k1_0000000000000000000000000000000000000000000000000000000000000000")
	if !errors.Is(err, service.ErrInvalidAPIKey) {
		t.Fatalf("expected ErrInvalidAPIKey, got %v", err)
	}
}
