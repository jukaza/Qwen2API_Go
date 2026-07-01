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
	"time"

	"github.com/redis/go-redis/v9"

	"qwen2api/internal/config"
)

type Session struct {
	Token     string    `json:"token"`
	IP        string    `json:"ip"`
	UserAgent string    `json:"user_agent"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

func (s Session) IsValid() bool {
	return time.Now().Before(s.ExpiresAt)
}

type SessionStore interface {
	GetSession(token string) (Session, error)
	SaveSession(session Session) error
	DeleteSession(token string) error
	ListSessions() ([]Session, error)
}

type memorySessionStore struct {
	mu       sync.RWMutex
	sessions map[string]Session
}

type fileSessionStore struct {
	path string
	mu   sync.Mutex
}

type redisSessionStore struct {
	client *redis.Client
}

func NewSessionStore(cfg config.Config) (SessionStore, error) {
	switch strings.ToLower(strings.TrimSpace(cfg.DataSaveMode)) {
	case "", "none", "guest":
		return &memorySessionStore{sessions: make(map[string]Session)}, nil
	case "file":
		return &fileSessionStore{path: filepath.Join("data", "sessions.json")}, nil
	case "redis":
		redisURL, err := redisURLFromConfig(cfg)
		if err != nil {
			return nil, err
		}
		client, err := newRedisClient(redisURL)
		if err != nil {
			return nil, err
		}
		return &redisSessionStore{client: client}, nil
	case "sqlite":
		db, err := newSQLiteDB(cfg)
		if err != nil {
			return nil, err
		}
		return &sqliteSessionStore{db: db}, nil
	default:
		return nil, errors.New("不支持的数据保存模式: " + cfg.DataSaveMode)
	}
}

// Memory implementation
func (s *memorySessionStore) GetSession(token string) (Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sess, ok := s.sessions[token]
	if !ok {
		return Session{}, errors.New("session not found")
	}
	return sess, nil
}

func (s *memorySessionStore) SaveSession(session Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[session.Token] = session
	return nil
}

func (s *memorySessionStore) DeleteSession(token string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, token)
	return nil
}

func (s *memorySessionStore) ListSessions() ([]Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]Session, 0, len(s.sessions))
	for _, sess := range s.sessions {
		result = append(result, sess)
	}
	return result, nil
}

// File implementation
func (s *fileSessionStore) GetSession(token string) (Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sessions, err := s.read()
	if err != nil {
		return Session{}, err
	}
	sess, ok := sessions[token]
	if !ok {
		return Session{}, errors.New("session not found")
	}
	return sess, nil
}

func (s *fileSessionStore) SaveSession(session Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	sessions, err := s.read()
	if err != nil {
		return err
	}
	sessions[session.Token] = session
	return s.write(sessions)
}

func (s *fileSessionStore) DeleteSession(token string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	sessions, err := s.read()
	if err != nil {
		return err
	}
	delete(sessions, token)
	return s.write(sessions)
}

func (s *fileSessionStore) ListSessions() ([]Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sessions, err := s.read()
	if err != nil {
		return nil, err
	}
	result := make([]Session, 0, len(sessions))
	for _, sess := range sessions {
		result = append(result, sess)
	}
	return result, nil
}

func (s *fileSessionStore) read() (map[string]Session, error) {
	if err := s.ensure(); err != nil {
		return nil, err
	}
	raw, err := os.ReadFile(s.path)
	if err != nil {
		return nil, err
	}
	var sessions map[string]Session
	if err := json.Unmarshal(raw, &sessions); err != nil {
		return nil, err
	}
	return sessions, nil
}

func (s *fileSessionStore) write(sessions map[string]Session) error {
	if err := s.ensure(); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(sessions, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, raw, 0644)
}

func (s *fileSessionStore) ensure() error {
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	if info, err := os.Stat(s.path); err == nil && info.Size() > 0 {
		return nil
	}
	raw, err := json.MarshalIndent(make(map[string]Session), "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, raw, 0644)
}

// Redis implementation
func (s *redisSessionStore) GetSession(token string) (Session, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	raw, err := s.client.Get(ctx, "session:"+token).Result()
	if errors.Is(err, redis.Nil) {
		return Session{}, errors.New("session not found")
	}
	if err != nil {
		return Session{}, err
	}

	var session Session
	if err := json.Unmarshal([]byte(raw), &session); err != nil {
		return Session{}, err
	}
	return session, nil
}

func (s *redisSessionStore) SaveSession(session Session) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	raw, err := json.Marshal(session)
	if err != nil {
		return err
	}

	ttl := time.Until(session.ExpiresAt)
	if ttl <= 0 {
		return nil
	}

	return s.client.Set(ctx, "session:"+session.Token, raw, ttl).Err()
}

func (s *redisSessionStore) DeleteSession(token string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s.client.Del(ctx, "session:"+token).Err()
}

func (s *redisSessionStore) ListSessions() ([]Session, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var cursor uint64
	result := make([]Session, 0)
	for {
		keys, next, err := s.client.Scan(ctx, cursor, "session:*", 100).Result()
		if err != nil {
			return nil, err
		}
		for _, key := range keys {
			raw, err := s.client.Get(ctx, key).Result()
			if err != nil {
				continue
			}
			var session Session
			if json.Unmarshal([]byte(raw), &session) == nil {
				result = append(result, session)
			}
		}
		cursor = next
		if cursor == 0 {
			break
		}
	}
	return result, nil
}

type sqliteSessionStore struct {
	db *sql.DB
}

func (s *sqliteSessionStore) GetSession(token string) (Session, error) {
	row := s.db.QueryRow(`
		SELECT ip, user_agent, created_at, expires_at
		FROM sessions WHERE token = ?
	`, token)

	var session Session
	session.Token = token
	var ip, userAgent sql.NullString
	var createdAt, expiresAt int64

	if err := row.Scan(&ip, &userAgent, &createdAt, &expiresAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Session{}, errors.New("session not found")
		}
		return Session{}, err
	}
	if ip.Valid {
		session.IP = ip.String
	}
	if userAgent.Valid {
		session.UserAgent = userAgent.String
	}
	session.CreatedAt = time.Unix(createdAt, 0)
	session.ExpiresAt = time.Unix(expiresAt, 0)

	return session, nil
}

func (s *sqliteSessionStore) SaveSession(session Session) error {
	_, err := s.db.Exec(`
		INSERT INTO sessions (token, ip, user_agent, created_at, expires_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(token) DO UPDATE SET
			ip=excluded.ip,
			user_agent=excluded.user_agent,
			created_at=excluded.created_at,
			expires_at=excluded.expires_at
	`, session.Token, session.IP, session.UserAgent, session.CreatedAt.Unix(), session.ExpiresAt.Unix())
	return err
}

func (s *sqliteSessionStore) DeleteSession(token string) error {
	_, err := s.db.Exec(`DELETE FROM sessions WHERE token = ?`, token)
	return err
}

func (s *sqliteSessionStore) ListSessions() ([]Session, error) {
	rows, err := s.db.Query(`SELECT token, ip, user_agent, created_at, expires_at FROM sessions`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []Session
	for rows.Next() {
		var session Session
		var ip, userAgent sql.NullString
		var createdAt, expiresAt int64
		if err := rows.Scan(&session.Token, &ip, &userAgent, &createdAt, &expiresAt); err != nil {
			return nil, err
		}
		if ip.Valid {
			session.IP = ip.String
		}
		if userAgent.Valid {
			session.UserAgent = userAgent.String
		}
		session.CreatedAt = time.Unix(createdAt, 0)
		session.ExpiresAt = time.Unix(expiresAt, 0)
		result = append(result, session)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if result == nil {
		result = []Session{}
	}
	return result, nil
}
