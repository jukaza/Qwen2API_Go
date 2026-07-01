package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/redis/go-redis/v9"

	"qwen2api/internal/config"
)

type APIKey struct {
	Key       string `json:"key"`
	Label     string `json:"label"`
	IsAdmin   bool   `json:"isAdmin"`
	CreatedAt int64  `json:"createdAt"`
}

type APIKeyStore interface {
	LoadAPIKeys() ([]APIKey, error)
	SaveAPIKey(key APIKey) error
	DeleteAPIKey(key string) error
	SaveAllAPIKeys(keys []APIKey) error
}

func NewAPIKeyStore(cfg config.Config) (APIKeyStore, error) {
	switch strings.ToLower(strings.TrimSpace(cfg.DataSaveMode)) {
	case "", "none", "guest":
		return &memAPIKeyStore{keys: []APIKey{}}, nil
	case "file":
		return &fileAPIKeyStore{path: filepath.Join("data", "data.json")}, nil
	case "redis":
		redisURL, err := redisURLFromConfig(cfg)
		if err != nil {
			return nil, err
		}
		client, err := newRedisClient(redisURL)
		if err != nil {
			return nil, err
		}
		return &redisAPIKeyStore{client: client}, nil
	case "sqlite":
		db, err := newSQLiteDB(cfg)
		if err != nil {
			return nil, err
		}
		return &sqliteAPIKeyStore{db: db}, nil
	default:
		return nil, errors.New("不支持的数据保存模式 (apikey): " + cfg.DataSaveMode)
	}
}

// Memory store
type memAPIKeyStore struct {
	keys []APIKey
	mu   sync.Mutex
}

func (s *memAPIKeyStore) LoadAPIKeys() ([]APIKey, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]APIKey(nil), s.keys...), nil
}

func (s *memAPIKeyStore) SaveAPIKey(key APIKey) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, k := range s.keys {
		if k.Key == key.Key {
			s.keys[i] = key
			return nil
		}
	}
	s.keys = append(s.keys, key)
	return nil
}

func (s *memAPIKeyStore) DeleteAPIKey(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	filtered := make([]APIKey, 0, len(s.keys))
	for _, k := range s.keys {
		if k.Key != key {
			filtered = append(filtered, k)
		}
	}
	s.keys = filtered
	return nil
}

func (s *memAPIKeyStore) SaveAllAPIKeys(keys []APIKey) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.keys = append([]APIKey(nil), keys...)
	return nil
}

// File store
type fileAPIKeyStore struct {
	path string
	mu   sync.Mutex
}

func (s *fileAPIKeyStore) LoadAPIKeys() ([]APIKey, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := s.read()
	if err != nil {
		return nil, err
	}
	if data.APIKeys == nil {
		return []APIKey{}, nil
	}
	return append([]APIKey(nil), data.APIKeys...), nil
}

func (s *fileAPIKeyStore) SaveAPIKey(key APIKey) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := s.read()
	if err != nil {
		return err
	}
	updated := false
	for i := range data.APIKeys {
		if data.APIKeys[i].Key == key.Key {
			data.APIKeys[i] = key
			updated = true
			break
		}
	}
	if !updated {
		data.APIKeys = append(data.APIKeys, key)
	}
	return s.write(data)
}

func (s *fileAPIKeyStore) DeleteAPIKey(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := s.read()
	if err != nil {
		return err
	}
	filtered := make([]APIKey, 0, len(data.APIKeys))
	for _, k := range data.APIKeys {
		if k.Key != key {
			filtered = append(filtered, k)
		}
	}
	data.APIKeys = filtered
	return s.write(data)
}

func (s *fileAPIKeyStore) SaveAllAPIKeys(keys []APIKey) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := s.read()
	if err != nil {
		return err
	}
	data.APIKeys = append([]APIKey(nil), keys...)
	return s.write(data)
}

func (s *fileAPIKeyStore) read() (FileData, error) {
	if err := s.ensure(); err != nil {
		return FileData{}, err
	}
	raw, err := os.ReadFile(s.path)
	if err != nil {
		return FileData{}, err
	}
	var data FileData
	if err := json.Unmarshal(raw, &data); err != nil {
		return FileData{}, err
	}
	return data, nil
}

func (s *fileAPIKeyStore) write(data FileData) error {
	if err := s.ensure(); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, raw, 0644)
}

func (s *fileAPIKeyStore) ensure() error {
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	if _, err := os.Stat(s.path); err == nil {
		return nil
	}
	defaultData := FileData{
		Accounts:   []Account{},
		ProxyPools: []ProxyPool{},
		APIKeys:    []APIKey{},
	}
	raw, err := json.MarshalIndent(defaultData, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, raw, 0644)
}

// Redis store
type redisAPIKeyStore struct {
	client *redis.Client
}

func (s *redisAPIKeyStore) LoadAPIKeys() ([]APIKey, error) {
	ctx, cancel := redisContext()
	defer cancel()

	keys, err := s.scanAPIKeyKeys(ctx)
	if err != nil {
		return nil, fmt.Errorf("redis scan apikeys failed: %w", err)
	}
	if len(keys) == 0 {
		return []APIKey{}, nil
	}

	pipe := s.client.Pipeline()
	cmds := make([]*redis.StringCmd, 0, len(keys))
	for _, key := range keys {
		cmds = append(cmds, pipe.Get(ctx, key))
	}
	if _, err := pipe.Exec(ctx); err != nil && !errors.Is(err, redis.Nil) {
		return nil, fmt.Errorf("redis pipe execute failed: %w", err)
	}

	apiKeys := make([]APIKey, 0, len(keys))
	for _, cmd := range cmds {
		val, err := cmd.Result()
		if err != nil && !errors.Is(err, redis.Nil) {
			continue
		}
		var key APIKey
		if err := json.Unmarshal([]byte(val), &key); err == nil {
			apiKeys = append(apiKeys, key)
		}
	}
	return apiKeys, nil
}

func (s *redisAPIKeyStore) SaveAPIKey(key APIKey) error {
	ctx, cancel := redisContext()
	defer cancel()

	raw, err := json.Marshal(key)
	if err != nil {
		return fmt.Errorf("failed to marshal apikey: %w", err)
	}
	err = s.client.Set(ctx, "apikey:"+key.Key, raw, 0).Err()
	if err != nil {
		return fmt.Errorf("failed to save apikey in redis: %w", err)
	}
	return nil
}

func (s *redisAPIKeyStore) DeleteAPIKey(key string) error {
	ctx, cancel := redisContext()
	defer cancel()
	err := s.client.Del(ctx, "apikey:"+key).Err()
	if err != nil {
		return fmt.Errorf("failed to delete apikey in redis: %w", err)
	}
	return nil
}

func (s *redisAPIKeyStore) SaveAllAPIKeys(keys []APIKey) error {
	ctx, cancel := redisContext()
	defer cancel()

	existingKeys, err := s.scanAPIKeyKeys(ctx)
	if err != nil {
		return err
	}

	pipe := s.client.TxPipeline()
	for _, key := range existingKeys {
		pipe.Del(ctx, key)
	}
	for _, k := range keys {
		raw, _ := json.Marshal(k)
		pipe.Set(ctx, "apikey:"+k.Key, raw, 0)
	}
	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to save all apikeys in redis: %w", err)
	}
	return nil
}

func (s *redisAPIKeyStore) scanAPIKeyKeys(ctx context.Context) ([]string, error) {
	var cursor uint64
	keys := make([]string, 0)
	for {
		batch, nextCursor, err := s.client.Scan(ctx, cursor, "apikey:*", 100).Result()
		if err != nil {
			return nil, err
		}
		keys = append(keys, batch...)
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}
	return keys, nil
}

// SQLite store
type sqliteAPIKeyStore struct {
	db *sql.DB
}

func (s *sqliteAPIKeyStore) LoadAPIKeys() ([]APIKey, error) {
	rows, err := s.db.Query("SELECT key, label, is_admin, created_at FROM api_keys")
	if err != nil {
		return nil, fmt.Errorf("failed to query api_keys: %w", err)
	}
	defer rows.Close()

	var keys []APIKey
	for rows.Next() {
		var k APIKey
		if err := rows.Scan(&k.Key, &k.Label, &k.IsAdmin, &k.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan api_key row: %w", err)
		}
		keys = append(keys, k)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error in api_keys: %w", err)
	}
	if keys == nil {
		keys = []APIKey{}
	}
	return keys, nil
}

func (s *sqliteAPIKeyStore) SaveAPIKey(key APIKey) error {
	_, err := s.db.Exec(`
		INSERT INTO api_keys (key, label, is_admin, created_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(key) DO UPDATE SET
			label = excluded.label,
			is_admin = excluded.is_admin,
			created_at = excluded.created_at
	`, key.Key, key.Label, key.IsAdmin, key.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to save api_key: %w", err)
	}
	return nil
}

func (s *sqliteAPIKeyStore) DeleteAPIKey(key string) error {
	_, err := s.db.Exec("DELETE FROM api_keys WHERE key = ?", key)
	if err != nil {
		return fmt.Errorf("failed to delete api_key: %w", err)
	}
	return nil
}

func (s *sqliteAPIKeyStore) SaveAllAPIKeys(keys []APIKey) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec("DELETE FROM api_keys"); err != nil {
		return fmt.Errorf("failed to truncate api_keys: %w", err)
	}

	stmt, err := tx.Prepare("INSERT INTO api_keys (key, label, is_admin, created_at) VALUES (?, ?, ?, ?)")
	if err != nil {
		return fmt.Errorf("failed to prepare insert api_key: %w", err)
	}
	defer stmt.Close()

	for _, k := range keys {
		if _, err := stmt.Exec(k.Key, k.Label, k.IsAdmin, k.CreatedAt); err != nil {
			return fmt.Errorf("failed to exec insert api_key: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit api_keys transaction: %w", err)
	}
	return nil
}
