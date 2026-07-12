package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/eternal-orbit-labs/gateway/internal/domain"
)

const apiKeyPrefix = "eol_k1_"

type APIKeyService struct {
	keys domain.PersonalAPIKeyRepository
}

func NewAPIKeyService(keys domain.PersonalAPIKeyRepository) *APIKeyService {
	return &APIKeyService{keys: keys}
}

type CreateKeyResult struct {
	Key        *domain.PersonalAPIKey `json:"key"`
	RawValue   string                 `json:"raw_value"`
}

func (s *APIKeyService) Create(ctx context.Context, userID, label string) (*CreateKeyResult, error) {
	rawBytes := make([]byte, 32)
	if _, err := rand.Read(rawBytes); err != nil {
		return nil, fmt.Errorf("generate key: %w", err)
	}
	rawHex := hex.EncodeToString(rawBytes)
	fullKey := apiKeyPrefix + rawHex

	hash := sha256.Sum256([]byte(fullKey))
	secureValue := hex.EncodeToString(hash[:])

	visiblePrefix := fullKey[:len(apiKeyPrefix)+8]

	key := &domain.PersonalAPIKey{
		UserID: userID,
		Label:  label,
		Prefix: visiblePrefix,
	}
	if err := s.keys.Create(ctx, key, secureValue); err != nil {
		return nil, err
	}

	return &CreateKeyResult{Key: key, RawValue: fullKey}, nil
}

func (s *APIKeyService) Authenticate(ctx context.Context, rawKey string) (*domain.PersonalAPIKey, error) {
	if !strings.HasPrefix(rawKey, apiKeyPrefix) {
		return nil, fmt.Errorf("invalid api key format")
	}

	hash := sha256.Sum256([]byte(rawKey))
	secureValue := hex.EncodeToString(hash[:])

	key, err := s.keys.GetBySecureValue(ctx, secureValue)
	if err != nil {
		return nil, fmt.Errorf("lookup key: %w", err)
	}
	if key == nil {
		return nil, fmt.Errorf("invalid api key")
	}

	// Fire-and-forget last_used_at update
	go s.keys.TouchLastUsed(context.Background(), key.ID)

	return key, nil
}

func (s *APIKeyService) List(ctx context.Context, userID string) ([]domain.PersonalAPIKey, error) {
	return s.keys.ListByUserID(ctx, userID)
}

func (s *APIKeyService) Delete(ctx context.Context, id, userID string) error {
	return s.keys.Delete(ctx, id, userID)
}
