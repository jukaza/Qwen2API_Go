package admin

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"qwen2api/internal/proxy"
	"qwen2api/internal/storage"
)

func (h *Handler) HandleGetProxyPools(w http.ResponseWriter, r *http.Request) {
	if h.proxyMgr == nil {
		writeJSON(w, http.StatusOK, map[string]any{"data": []storage.ProxyPool{}})
		return
	}
	proxies, err := h.proxyStore.LoadProxies()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": proxies})
}

func (h *Handler) HandleAddProxy(w http.ResponseWriter, r *http.Request) {
	var payload storage.ProxyPool
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "Invalid request payload"})
		return
	}
	payload.ID = fmt.Sprintf("proxy_%d", time.Now().UnixNano())
	payload.CreatedAt = time.Now().UnixMilli()
	if payload.Type == "" {
		payload.Type = "http" // default type
	}

	if err := h.proxyStore.SaveProxy(payload); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"message": "Proxy added successfully", "data": payload})
}

func (h *Handler) HandleUpdateProxy(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "Proxy ID is required"})
		return
	}
	id := parts[len(parts)-1]

	var payload storage.ProxyPool
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "Invalid request payload"})
		return
	}
	payload.ID = id // ensure ID is preserved

	if err := h.proxyStore.SaveProxy(payload); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"message": "Proxy updated successfully", "data": payload})
}

func (h *Handler) HandleDeleteProxy(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "Proxy ID is required"})
		return
	}
	id := parts[len(parts)-1]

	if err := h.proxyStore.DeleteProxy(id); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"message": "Proxy deleted successfully"})
}

func (h *Handler) HandleTestProxy(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 5 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "Proxy ID is required"})
		return
	}
	id := parts[len(parts)-2] // /api/proxy-pools/{id}/test

	proxies, err := h.proxyStore.LoadProxies()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	var targetProxy *storage.ProxyPool
	for i := range proxies {
		if proxies[i].ID == id {
			targetProxy = &proxies[i]
			break
		}
	}

	if targetProxy == nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "Proxy not found"})
		return
	}

	ok, msg := h.proxyMgr.TestProxy(r.Context(), *targetProxy)

	targetProxy.TestStatus = "dead"
	targetProxy.LastError = msg
	if ok {
		targetProxy.TestStatus = "ok"
		targetProxy.LastError = ""
	}
	targetProxy.LastTestedAt = time.Now().UnixMilli()

	h.proxyStore.SaveProxy(*targetProxy)

	if ok {
		writeJSON(w, http.StatusOK, map[string]any{"message": "Proxy is healthy", "data": targetProxy})
	} else {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "Proxy test failed", "message": msg, "data": targetProxy})
	}
}

func (h *Handler) HandleVercelDeploy(w http.ResponseWriter, r *http.Request) {
	var payload proxy.DeployVercelRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "Invalid payload"})
		return
	}

	results, err := proxy.DeployVercel(r.Context(), payload)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error(), "results": results})
		return
	}

	// Add deployed proxies to the pool automatically
	for _, res := range results {
		newProxy := storage.ProxyPool{
			ID:         fmt.Sprintf("proxy_%d_%s", time.Now().UnixNano(), res.ProjectName),
			Name:       res.ProjectName,
			ProxyURL:   res.DeployURL,
			Type:       "vercel",
			IsActive:   true,
			TestStatus: "ok",
			CreatedAt:  time.Now().UnixMilli(),
		}
		_ = h.proxyStore.SaveProxy(newProxy)
	}

	writeJSON(w, http.StatusOK, map[string]any{"message": "Deployments completed", "data": results})
}

func (h *Handler) HandleAutoBindProxies(w http.ResponseWriter, r *http.Request) {
	if h.proxyMgr == nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "Proxy Manager is not initialized"})
		return
	}

	if err := h.proxyMgr.AutoBind(); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "Failed to auto-bind: " + err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"message": "Auto-bind completed successfully"})
}

func (h *Handler) HandleVercelClean(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		APIToken    string `json:"apiToken"`
		ProjectName string `json:"projectName"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "Invalid payload"})
		return
	}

	if payload.APIToken == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "API Token is required"})
		return
	}

	deletedCount, err := proxy.CleanVercelProjects(r.Context(), payload.APIToken)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"message":      fmt.Sprintf("Cleaned up %d Vercel projects", deletedCount),
		"deletedCount": deletedCount,
	})
}
