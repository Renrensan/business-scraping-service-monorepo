package session

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"business-scraping-service/backend/internal/model"
	"business-scraping-service/backend/internal/utils"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	sessionKeyPrefix   = "bizscrape:session:"
	userSessionsPrefix = "bizscrape:user_sessions:"
	DefaultSessionTTL  = 24 * time.Hour
)

type Store interface {
	CreateSession(ctx context.Context, data model.SessionData) (string, error)
	GetSession(ctx context.Context, token string) (*model.SessionData, error)
	TouchSession(ctx context.Context, token string) error
	DeleteSession(ctx context.Context, token string) error
	DeleteAllUserSessions(ctx context.Context, adminID uuid.UUID) error
}

type RedisStore struct {
	rdb *redis.Client
	ttl time.Duration
}

func NewRedisStore(rdb *redis.Client, ttl time.Duration) *RedisStore {
	if ttl <= 0 {
		ttl = DefaultSessionTTL
	}
	return &RedisStore{
		rdb: rdb,
		ttl: ttl,
	}
}

func (s *RedisStore) CreateSession(ctx context.Context, data model.SessionData) (string, error) {
	token, err := utils.GenerateRandomToken(32)
	if err != nil {
		return "", fmt.Errorf("failed to generate session token: %w", err)
	}

	data.Token = token
	data.CreatedAt = time.Now().UTC()
	data.LastActiveAt = data.CreatedAt

	payload, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("failed to marshal session data: %w", err)
	}

	sessionKey := sessionKeyPrefix + token
	userKey := userSessionsPrefix + data.AdminID.String()

	pipe := s.rdb.TxPipeline()
	pipe.Set(ctx, sessionKey, payload, s.ttl)
	pipe.SAdd(ctx, userKey, token)
	pipe.Expire(ctx, userKey, s.ttl*7)

	if _, err := pipe.Exec(ctx); err != nil {
		return "", fmt.Errorf("failed to save session in redis: %w", err)
	}

	return token, nil
}

func (s *RedisStore) GetSession(ctx context.Context, token string) (*model.SessionData, error) {
	if token == "" {
		return nil, fmt.Errorf("empty session token")
	}

	sessionKey := sessionKeyPrefix + token
	val, err := s.rdb.Get(ctx, sessionKey).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // expired / not found
		}
		return nil, fmt.Errorf("redis error getting session: %w", err)
	}

	var data model.SessionData
	if err := json.Unmarshal([]byte(val), &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session: %w", err)
	}

	return &data, nil
}

func (s *RedisStore) TouchSession(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}
	sessionKey := sessionKeyPrefix + token
	return s.rdb.Expire(ctx, sessionKey, s.ttl).Err()
}

func (s *RedisStore) DeleteSession(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}

	session, err := s.GetSession(ctx, token)
	if err != nil {
		return err
	}
	if session == nil {
		return nil
	}

	sessionKey := sessionKeyPrefix + token
	userKey := userSessionsPrefix + session.AdminID.String()

	pipe := s.rdb.TxPipeline()
	pipe.Del(ctx, sessionKey)
	pipe.SRem(ctx, userKey, token)

	_, err = pipe.Exec(ctx)
	return err
}

func (s *RedisStore) DeleteAllUserSessions(ctx context.Context, adminID uuid.UUID) error {
	userKey := userSessionsPrefix + adminID.String()

	tokens, err := s.rdb.SMembers(ctx, userKey).Result()
	if err != nil && err != redis.Nil {
		return fmt.Errorf("failed to get user sessions: %w", err)
	}

	if len(tokens) == 0 {
		return nil
	}

	pipe := s.rdb.TxPipeline()
	for _, token := range tokens {
		pipe.Del(ctx, sessionKeyPrefix+token)
	}
	pipe.Del(ctx, userKey)

	_, err = pipe.Exec(ctx)
	return err
}
