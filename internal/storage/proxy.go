package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/redis/go-redis/v9"

	"qwen2api/internal/config"
)

type ProxyPool struct {
	ID                   string `json:"id"`
	Name                 string `json:"name"`
	ProxyURL             string `json:"proxyUrl"`
	NoProxy              string `json:"noProxy,omitempty"`
	Type                 string `json:"type,omitempty"` // "http", "vercel", "cloudflare", "deno"
	IsActive             bool   `json:"isActive"`
	StrictProxy          bool   `json:"strictProxy"`
	TestStatus           string `json:"testStatus,omitempty"`
	LastError            string `json:"lastError,omitempty"`
	LastTestedAt         int64  `json:"lastTestedAt,omitempty"`
	CreatedAt            int64  `json:"createdAt"`
	BoundConnectionCount int    `json:"boundConnectionCount,omitempty"`
}

type ProxyStore interface {
	LoadProxies() ([]ProxyPool, error)
	SaveProxy(proxy ProxyPool) error
	DeleteProxy(id string) error
	SaveAllProxies(proxies []ProxyPool) error
}

func NewProxyStore(cfg config.Config) (ProxyStore, error) {
	switch strings.ToLower(strings.TrimSpace(cfg.DataSaveMode)) {
	case "", "none", "guest":
		return &memProxyStore{proxies: []ProxyPool{}}, nil
	case "file":
		return &fileProxyStore{path: filepath.Join("data", "data.json")}, nil
	case "redis":
		redisURL, err := redisURLFromConfig(cfg)
		if err != nil {
			return nil, err
		}
		client, err := newRedisClient(redisURL)
		if err != nil {
			return nil, err
		}
		return &redisProxyStore{client: client}, nil
	case "sqlite":
		db, err := newSQLiteDB(cfg)
		if err != nil {
			return nil, err
		}
		return &sqliteProxyStore{db: db}, nil
	default:
		return nil, errors.New("不支持的数据保存模式 (proxy): " + cfg.DataSaveMode)
	}
}

// Memory store for none/guest mode
type memProxyStore struct {
	proxies []ProxyPool
	mu      sync.Mutex
}

func (s *memProxyStore) LoadProxies() ([]ProxyPool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]ProxyPool(nil), s.proxies...), nil
}

func (s *memProxyStore) SaveProxy(proxy ProxyPool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, p := range s.proxies {
		if p.ID == proxy.ID {
			s.proxies[i] = proxy
			return nil
		}
	}
	s.proxies = append(s.proxies, proxy)
	return nil
}

func (s *memProxyStore) DeleteProxy(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	filtered := make([]ProxyPool, 0, len(s.proxies))
	for _, p := range s.proxies {
		if p.ID != id {
			filtered = append(filtered, p)
		}
	}
	s.proxies = filtered
	return nil
}

func (s *memProxyStore) SaveAllProxies(proxies []ProxyPool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.proxies = append([]ProxyPool(nil), proxies...)
	return nil
}

// File store (reuses FileData structure from accounts.go)
type fileProxyStore struct {
	path string
	mu   sync.Mutex
}

func (s *fileProxyStore) LoadProxies() ([]ProxyPool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := s.read()
	if err != nil {
		return nil, err
	}
	return append([]ProxyPool(nil), data.ProxyPools...), nil
}

func (s *fileProxyStore) SaveProxy(proxy ProxyPool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := s.read()
	if err != nil {
		return err
	}
	updated := false
	for i := range data.ProxyPools {
		if data.ProxyPools[i].ID == proxy.ID {
			data.ProxyPools[i] = proxy
			updated = true
			break
		}
	}
	if !updated {
		data.ProxyPools = append(data.ProxyPools, proxy)
	}
	return s.write(data)
}

func (s *fileProxyStore) DeleteProxy(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := s.read()
	if err != nil {
		return err
	}
	filtered := make([]ProxyPool, 0, len(data.ProxyPools))
	for _, p := range data.ProxyPools {
		if p.ID != id {
			filtered = append(filtered, p)
		}
	}
	data.ProxyPools = filtered
	return s.write(data)
}

func (s *fileProxyStore) SaveAllProxies(proxies []ProxyPool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := s.read()
	if err != nil {
		return err
	}
	data.ProxyPools = append([]ProxyPool(nil), proxies...)
	return s.write(data)
}

func (s *fileProxyStore) read() (FileData, error) {
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

func (s *fileProxyStore) write(data FileData) error {
	if err := s.ensure(); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, raw, 0644)
}

func (s *fileProxyStore) ensure() error {
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
	}
	raw, err := json.MarshalIndent(defaultData, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, raw, 0644)
}

// Redis store
type redisProxyStore struct {
	client *redis.Client
}

func (s *redisProxyStore) LoadProxies() ([]ProxyPool, error) {
	ctx, cancel := redisContext()
	defer cancel()

	keys, err := s.scanProxyKeys(ctx)
	if err != nil {
		return nil, err
	}
	if len(keys) == 0 {
		return []ProxyPool{}, nil
	}

	pipe := s.client.Pipeline()
	cmds := make([]*redis.StringCmd, 0, len(keys))
	for _, key := range keys {
		cmds = append(cmds, pipe.Get(ctx, key))
	}
	if _, err := pipe.Exec(ctx); err != nil && !errors.Is(err, redis.Nil) {
		return nil, err
	}

	proxies := make([]ProxyPool, 0, len(keys))
	for _, cmd := range cmds {
		val, err := cmd.Result()
		if err != nil && !errors.Is(err, redis.Nil) {
			continue
		}
		var proxy ProxyPool
		if err := json.Unmarshal([]byte(val), &proxy); err == nil {
			proxies = append(proxies, proxy)
		}
	}
	return proxies, nil
}

func (s *redisProxyStore) SaveProxy(proxy ProxyPool) error {
	ctx, cancel := redisContext()
	defer cancel()

	raw, err := json.Marshal(proxy)
	if err != nil {
		return err
	}
	return s.client.Set(ctx, "proxy:"+proxy.ID, raw, 0).Err()
}

func (s *redisProxyStore) DeleteProxy(id string) error {
	ctx, cancel := redisContext()
	defer cancel()
	return s.client.Del(ctx, "proxy:"+id).Err()
}

func (s *redisProxyStore) SaveAllProxies(proxies []ProxyPool) error {
	ctx, cancel := redisContext()
	defer cancel()

	keys, err := s.scanProxyKeys(ctx)
	if err != nil {
		return err
	}

	pipe := s.client.TxPipeline()
	for _, key := range keys {
		pipe.Del(ctx, key)
	}
	for _, p := range proxies {
		raw, _ := json.Marshal(p)
		pipe.Set(ctx, "proxy:"+p.ID, raw, 0)
	}
	_, err = pipe.Exec(ctx)
	return err
}

func (s *redisProxyStore) scanProxyKeys(ctx context.Context) ([]string, error) {
	var cursor uint64
	keys := make([]string, 0)
	for {
		batch, nextCursor, err := s.client.Scan(ctx, cursor, "proxy:*", 100).Result()
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

type sqliteProxyStore struct {
	db *sql.DB
}

func (s *sqliteProxyStore) LoadProxies() ([]ProxyPool, error) {
	rows, err := s.db.Query(`
		SELECT id, name, proxy_url, no_proxy, type, is_active, strict_proxy,
		       test_status, last_error, last_tested_at, created_at, bound_connection_count
		FROM proxies
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var proxies []ProxyPool
	for rows.Next() {
		var p ProxyPool
		var noProxy, pType, testStatus, lastError sql.NullString
		var lastTestedAt sql.NullInt64
		if err := rows.Scan(
			&p.ID, &p.Name, &p.ProxyURL, &noProxy, &pType, &p.IsActive, &p.StrictProxy,
			&testStatus, &lastError, &lastTestedAt, &p.CreatedAt, &p.BoundConnectionCount,
		); err != nil {
			return nil, err
		}
		if noProxy.Valid {
			p.NoProxy = noProxy.String
		}
		if pType.Valid {
			p.Type = pType.String
		}
		if testStatus.Valid {
			p.TestStatus = testStatus.String
		}
		if lastError.Valid {
			p.LastError = lastError.String
		}
		if lastTestedAt.Valid {
			p.LastTestedAt = lastTestedAt.Int64
		}
		proxies = append(proxies, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if proxies == nil {
		proxies = []ProxyPool{}
	}
	return proxies, nil
}

func (s *sqliteProxyStore) SaveProxy(proxy ProxyPool) error {
	_, err := s.db.Exec(`
		INSERT INTO proxies (
			id, name, proxy_url, no_proxy, type, is_active, strict_proxy,
			test_status, last_error, last_tested_at, created_at, bound_connection_count
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			name=excluded.name,
			proxy_url=excluded.proxy_url,
			no_proxy=excluded.no_proxy,
			type=excluded.type,
			is_active=excluded.is_active,
			strict_proxy=excluded.strict_proxy,
			test_status=excluded.test_status,
			last_error=excluded.last_error,
			last_tested_at=excluded.last_tested_at,
			created_at=excluded.created_at,
			bound_connection_count=excluded.bound_connection_count
	`, proxy.ID, proxy.Name, proxy.ProxyURL, proxy.NoProxy, proxy.Type, proxy.IsActive, proxy.StrictProxy,
		proxy.TestStatus, proxy.LastError, proxy.LastTestedAt, proxy.CreatedAt, proxy.BoundConnectionCount)
	return err
}

func (s *sqliteProxyStore) DeleteProxy(id string) error {
	_, err := s.db.Exec(`DELETE FROM proxies WHERE id = ?`, id)
	return err
}

func (s *sqliteProxyStore) SaveAllProxies(proxies []ProxyPool) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM proxies`); err != nil {
		return err
	}

	stmt, err := tx.Prepare(`
		INSERT INTO proxies (
			id, name, proxy_url, no_proxy, type, is_active, strict_proxy,
			test_status, last_error, last_tested_at, created_at, bound_connection_count
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, p := range proxies {
		if _, err := stmt.Exec(
			p.ID, p.Name, p.ProxyURL, p.NoProxy, p.Type, p.IsActive, p.StrictProxy,
			p.TestStatus, p.LastError, p.LastTestedAt, p.CreatedAt, p.BoundConnectionCount,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}
