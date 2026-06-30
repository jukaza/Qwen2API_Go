package storage

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestSessionStoresHaveConsistentSemantics(t *testing.T) {
	memStore := &memorySessionStore{sessions: make(map[string]Session)}
	fileStore := &fileSessionStore{path: filepath.Join(t.TempDir(), "sessions.json")}

	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis.Run() error = %v", err)
	}
	defer mr.Close()

	redisStore := &redisSessionStore{
		client: redis.NewClient(&redis.Options{Addr: mr.Addr()}),
	}

	stores := map[string]SessionStore{
		"memory": memStore,
		"file":   fileStore,
		"redis":  redisStore,
	}

	for name, store := range stores {
		t.Run(name, func(t *testing.T) {
			s1 := Session{
				Token:     "sess-1",
				IP:        "127.0.0.1",
				UserAgent: "Mozilla/5.0",
				CreatedAt: time.Now(),
				ExpiresAt: time.Now().Add(1 * time.Hour),
			}
			s2 := Session{
				Token:     "sess-2",
				IP:        "192.168.1.1",
				UserAgent: "Safari/537.36",
				CreatedAt: time.Now(),
				ExpiresAt: time.Now().Add(2 * time.Hour),
			}

			// Save and verify Get
			if err := store.SaveSession(s1); err != nil {
				t.Fatalf("SaveSession s1 error = %v", err)
			}
			if err := store.SaveSession(s2); err != nil {
				t.Fatalf("SaveSession s2 error = %v", err)
			}

			got1, err := store.GetSession(s1.Token)
			if err != nil {
				t.Fatalf("GetSession s1 error = %v", err)
			}
			if got1.Token != s1.Token || got1.IP != s1.IP || got1.UserAgent != s1.UserAgent {
				t.Fatalf("GetSession s1 mismatch: got %+v, want %+v", got1, s1)
			}

			// List sessions
			list, err := store.ListSessions()
			if err != nil {
				t.Fatalf("ListSessions error = %v", err)
			}
			if len(list) != 2 {
				t.Fatalf("ListSessions returned len = %d, want 2", len(list))
			}

			// Delete session
			if err := store.DeleteSession(s1.Token); err != nil {
				t.Fatalf("DeleteSession s1 error = %v", err)
			}

			_, err = store.GetSession(s1.Token)
			if err == nil {
				t.Fatalf("expected s1 to be deleted, but GetSession succeeded")
			}

			list, err = store.ListSessions()
			if err != nil {
				t.Fatalf("ListSessions after delete error = %v", err)
			}
			if len(list) != 1 {
				t.Fatalf("ListSessions after delete returned len = %d, want 1", len(list))
			}
		})
	}
}
