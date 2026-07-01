package proxy

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"sync"
	"time"

	"qwen2api/internal/logging"
	"qwen2api/internal/storage"
)

type Manager struct {
	proxyStore   storage.ProxyStore
	accountStore storage.AccountStore
	logger       *logging.Logger
	mu           sync.RWMutex
}

func NewManager(proxyStore storage.ProxyStore, accountStore storage.AccountStore, logger *logging.Logger) *Manager {
	return &Manager{
		proxyStore:   proxyStore,
		accountStore: accountStore,
		logger:       logger,
	}
}

// GetProxyForAccount returns the ProxyPool that is pinned to the given account email.
// If the account has no proxy assigned or the assigned proxy is inactive, it will attempt to find a fallback.
func (m *Manager) GetProxyForAccount(email string) *storage.ProxyPool {
	accounts, err := m.accountStore.LoadAccounts()
	if err != nil {
		return nil
	}

	var proxyID string
	for _, acc := range accounts {
		if acc.Email == email {
			proxyID = acc.ProxyID
			break
		}
	}

	proxies, err := m.proxyStore.LoadProxies()
	if err != nil {
		return nil
	}

	var activeProxies []storage.ProxyPool
	for _, p := range proxies {
		if p.IsActive {
			activeProxies = append(activeProxies, p)
		}
	}

	if len(activeProxies) == 0 {
		return nil
	}

	// Find the assigned proxy
	if proxyID != "" {
		for i, p := range activeProxies {
			if p.ID == proxyID {
				return &activeProxies[i]
			}
		}
	}

	// Fallback to random active proxy
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	randomProxy := activeProxies[r.Intn(len(activeProxies))]
	return &randomProxy
}

// AutoBind optimally distributes all active accounts across all active proxies
func (m *Manager) AutoBind() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	proxies, err := m.proxyStore.LoadProxies()
	if err != nil {
		return err
	}

	var activeProxies []storage.ProxyPool
	for _, p := range proxies {
		if p.IsActive {
			activeProxies = append(activeProxies, p)
		}
	}

	if len(activeProxies) == 0 {
		return fmt.Errorf("no active proxies available")
	}

	accounts, err := m.accountStore.LoadAccounts()
	if err != nil {
		return err
	}

	if len(accounts) == 0 {
		return nil // nothing to bind
	}

	// Shuffle proxies to avoid deterministic bias
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	r.Shuffle(len(activeProxies), func(i, j int) {
		activeProxies[i], activeProxies[j] = activeProxies[j], activeProxies[i]
	})

	proxyCount := len(activeProxies)
	for i := range accounts {
		assignedProxy := activeProxies[i%proxyCount]
		accounts[i].ProxyID = assignedProxy.ID
	}

	// Update bound connections count
	for i := range proxies {
		count := 0
		for _, acc := range accounts {
			if acc.ProxyID == proxies[i].ID {
				count++
			}
		}
		proxies[i].BoundConnectionCount = count
	}

	if err := m.proxyStore.SaveAllProxies(proxies); err != nil {
		return err
	}

	return m.accountStore.SaveAllAccounts(accounts)
}

// TestProxy checks if the proxy is healthy by connecting through it
func (m *Manager) TestProxy(ctx context.Context, proxy storage.ProxyPool) (bool, string) {
	// If it's a Vercel/Cloudflare relay proxy, we test it via its endpoint logic
	if proxy.Type == "vercel" || proxy.Type == "cloudflare" || proxy.Type == "deno" {
		return m.testCloudRelay(ctx, proxy)
	}
	return m.testHttpProxy(ctx, proxy)
}

func (m *Manager) testCloudRelay(ctx context.Context, proxy storage.ProxyPool) (bool, string) {
	req, err := http.NewRequestWithContext(ctx, "GET", proxy.ProxyURL, nil)
	if err != nil {
		return false, fmt.Sprintf("failed to create request: %v", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Sprintf("connection failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 500 {
		return false, fmt.Sprintf("server error: %d", resp.StatusCode)
	}

	return true, "ok"
}

func (m *Manager) testHttpProxy(ctx context.Context, proxy storage.ProxyPool) (bool, string) {
	proxyURL, err := url.Parse(proxy.ProxyURL)
	if err != nil {
		return false, fmt.Sprintf("invalid proxy url: %v", err)
	}

	transport := &http.Transport{
		Proxy: http.ProxyURL(proxyURL),
	}
	client := &http.Client{
		Transport: transport,
		Timeout:   10 * time.Second,
	}

	// Ping a reliable endpoint
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.openai.com/v1/models", nil)
	if err != nil {
		return false, fmt.Sprintf("request error: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Sprintf("proxy connection failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 { // even 401 Unauthorized means the connection succeeded
		return true, "ok"
	}

	return false, fmt.Sprintf("unexpected status: %d", resp.StatusCode)
}

// CheckAllHealth runs background health checks
func (m *Manager) CheckAllHealth(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	proxies, err := m.proxyStore.LoadProxies()
	if err != nil {
		return err
	}

	changed := false
	for i, p := range proxies {
		if !p.IsActive && p.TestStatus == "dead" {
			continue // Skip checking already dead proxies to save time, or we can check them occasionally.
		}

		ok, msg := m.TestProxy(ctx, p)
		proxies[i].LastTestedAt = time.Now().UnixMilli()
		if ok {
			proxies[i].TestStatus = "ok"
			proxies[i].LastError = ""
		} else {
			proxies[i].TestStatus = "dead"
			proxies[i].LastError = msg
			// Option: we could auto-disable here if we want
			// proxies[i].IsActive = false
		}
		changed = true
	}

	if changed {
		return m.proxyStore.SaveAllProxies(proxies)
	}
	return nil
}
