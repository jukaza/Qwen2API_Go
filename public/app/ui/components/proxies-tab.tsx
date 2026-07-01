"use client";

import { useState, useEffect } from "react";
import { useTranslation } from "react-i18next";
import { Input } from "@heroui/react";
import { RefreshCw, Search, Plus, Trash2, Zap, Play, Activity, CheckCircle2, XCircle, ExternalLink } from "lucide-react";
import type { ProxyPoolItem, ProxyPoolsResponse } from "../types";
import { apiRequest } from "../api";
import { SectionTitle } from "./primitives";
import { ProxyAddModal } from "./proxy-add-modal";

export function ProxiesTab({ apiKey }: { apiKey: string }) {
  const { t } = useTranslation();
  const [proxies, setProxies] = useState<ProxyPoolItem[]>([]);
  const [loading, setLoading] = useState(false);
  const [keyword, setKeyword] = useState("");
  const [deploying, setDeploying] = useState(false);
  const [vercelToken, setVercelToken] = useState("");
  const [vercelProject, setVercelProject] = useState("qwen-relay");
  const [deployCount, setDeployCount] = useState(1);
  const [addModalOpen, setAddModalOpen] = useState(false);

  const fetchProxies = async () => {
    setLoading(true);
    try {
      const res = await apiRequest<ProxyPoolsResponse>("/api/proxy-pools", {}, apiKey);
      if (res && res.data) {
        setProxies(res.data);
      }
    } catch (e) {
      console.error(e);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (apiKey) {
      fetchProxies();
    }
  }, [apiKey]);

  const handleTest = async (id: string) => {
    try {
      await apiRequest(`/api/proxy-pools/test/${id}`, { method: "POST" }, apiKey);
      fetchProxies();
    } catch (e) {
      console.error(e);
      fetchProxies(); // refresh to show error status
    }
  };

  const handleDelete = async (id: string) => {
    if (!confirm("Are you sure you want to delete this proxy?")) return;
    try {
      await apiRequest(`/api/proxy-pools/delete/${id}`, { method: "DELETE" }, apiKey);
      fetchProxies();
    } catch (e) {
      console.error(e);
    }
  };

  const handleAddSubmit = async (name: string, url: string, type: string) => {
    try {
      await apiRequest("/api/proxy-pools/add", {
        method: "POST",
        body: JSON.stringify({
          name: name || "Custom Proxy",
          proxyUrl: url,
          type: type || (url.includes("vercel") ? "vercel" : "http"),
          isActive: true,
          strictProxy: false,
        }),
      }, apiKey);
      fetchProxies();
    } catch (e) {
      console.error(e);
      alert("Failed to add proxy");
    }
  };

  const handleBulkAdd = async (lines: string) => {
    const proxies = lines.split("\n").map(l => l.trim()).filter(Boolean);
    let added = 0;
    for (const line of proxies) {
      const parts = line.split(" ");
      const url = parts[0];
      const name = parts.slice(1).join(" ") || "Bulk Proxy";
      try {
        await apiRequest("/api/proxy-pools/add", {
          method: "POST",
          body: JSON.stringify({
            name,
            proxyUrl: url,
            type: url.startsWith("socks") ? "socks5" : "http",
            isActive: true,
            strictProxy: false,
          }),
        }, apiKey);
        added++;
      } catch (e) {
        console.error(e);
      }
    }
    alert(`Added ${added} proxies`);
    fetchProxies();
  };

  const handleAutoBind = async () => {
    if (!confirm("This will redistribute all accounts across active proxies. Continue?")) return;
    try {
      await apiRequest("/api/proxy-pools/auto-bind", { method: "POST" }, apiKey);
      alert("Auto-bind completed");
      fetchProxies();
    } catch (e) {
      console.error(e);
      alert("Failed to auto-bind");
    }
  };

  const handleVercelDeploy = async () => {
    if (!vercelToken) {
      alert("Vercel token is required");
      return;
    }
    setDeploying(true);
    try {
      await apiRequest("/api/proxy-pools/vercel-deploy", {
        method: "POST",
        body: JSON.stringify({
          vercelToken: vercelToken,
          projectName: vercelProject || "qwen-relay",
          regions: ["sin1"], // default to Singapore
          count: deployCount,
        }),
      }, apiKey);
      alert("Deployment started. Proxies will be added automatically.");
      fetchProxies();
    } catch (e) {
      console.error(e);
      alert("Deploy failed");
    } finally {
      setDeploying(false);
    }
  };

  const handleVercelClean = async () => {
    if (!vercelToken) {
      alert("Vercel token is required to clean projects");
      return;
    }
    if (!confirm("DANGER: This will delete ALL Vercel projects in the account associated with this token. Are you absolutely sure?")) return;
    
    setDeploying(true);
    try {
      const res = await apiRequest<{message: string, deletedCount: number}>("/api/proxy-pools/vercel-clean", {
        method: "POST",
        body: JSON.stringify({
          apiToken: vercelToken,
        }),
      }, apiKey);
      if (res) {
        alert(res.message);
      }
    } catch (e) {
      console.error(e);
      alert("Clean up failed");
    } finally {
      setDeploying(false);
    }
  };

  const filtered = proxies.filter(
    (p) =>
      p.name.toLowerCase().includes(keyword.toLowerCase()) ||
      p.proxyUrl.toLowerCase().includes(keyword.toLowerCase())
  );

  return (
    <div className="admin-card">
      <div className="admin-card-header">
        <SectionTitle
          title="Proxy & Relays"
          description="Manage proxies and edge relays for IP rotation"
          action={
            <div className="flex flex-wrap items-center justify-end gap-2">
              <div className="relative">
                <Search size={14} className="absolute left-3 top-1/2 -translate-y-1/2 text-[var(--text-muted)]" />
                <Input
                  placeholder="Search proxy..."
                  value={keyword}
                  onChange={(e) => setKeyword(e.target.value)}
                  className="w-48 pl-9"
                />
              </div>
              <button
                className="admin-btn admin-btn-ghost"
                onClick={() => setAddModalOpen(true)}
                title="Add proxy"
              >
                <Plus size={16} />
                Add
              </button>
              <button
                className="admin-btn admin-btn-secondary"
                onClick={handleAutoBind}
                title="Re-balance accounts to proxies"
              >
                <Zap size={16} />
                Auto Bind
              </button>
              <button
                className="admin-btn admin-btn-primary"
                disabled={loading}
                onClick={() => void fetchProxies()}
                title="Refresh"
              >
                <RefreshCw size={16} className={loading ? "animate-spin" : ""} />
                Refresh
              </button>
            </div>
          }
        />
      </div>

      <div className="admin-card-body">
        {/* Deploy panel */}
        <div className="p-4 bg-[var(--card-bg)] border border-[var(--border)] rounded-xl mb-6">
          <div className="flex items-center justify-between mb-3">
            <h4 className="font-semibold text-sm flex items-center gap-2">
              <Play size={16} /> Bulk Vercel Deploy
            </h4>
          </div>
          <div className="flex flex-wrap items-end gap-3">
            <div className="flex-1 min-w-[200px]">
              <label className="text-xs text-[var(--text-secondary)] mb-1 flex items-center justify-between">
                Vercel API Token
                <a href="https://vercel.com/account/tokens" target="_blank" rel="noreferrer" className="text-[var(--primary)] hover:underline flex items-center gap-1 text-[10px]">
                  Get Token <ExternalLink size={10} />
                </a>
              </label>
              <Input 
                placeholder="vt_..." 
                type="password" 
                value={vercelToken}
                onChange={(e) => setVercelToken(e.target.value)}
              />
            </div>
            <div className="flex-1 min-w-[150px]">
              <label className="text-xs text-[var(--text-secondary)] mb-1 block">Project Name Prefix</label>
              <Input 
                placeholder="qwen-relay" 
                value={vercelProject}
                onChange={(e) => setVercelProject(e.target.value)}
              />
            </div>
            <div className="w-[100px]">
              <label className="text-xs text-[var(--text-secondary)] mb-1 block">Count (Max 10)</label>
              <Input 
                type="number"
                min={1}
                max={10}
                value={deployCount.toString()}
                onChange={(e) => setDeployCount(parseInt(e.target.value) || 1)}
              />
            </div>
            <button 
              className="admin-btn admin-btn-primary h-10"
              onClick={handleVercelDeploy}
              disabled={deploying || !vercelToken}
            >
              {deploying ? <RefreshCw size={16} className="animate-spin" /> : <Play size={16} />}
              Deploy
            </button>
            <button 
              className="admin-btn admin-btn-danger h-10"
              onClick={handleVercelClean}
              disabled={deploying || !vercelToken}
              title="Delete all Vercel projects in this account"
            >
              {deploying ? <RefreshCw size={16} className="animate-spin" /> : <Trash2 size={16} />}
              Clean All
            </button>
          </div>
        </div>

        <div className="admin-model-grid">
          {filtered.map((proxy) => (
            <div className="admin-model-card relative" key={proxy.id}>
              <div className="flex items-start justify-between gap-3 mb-3">
                <div className="min-w-0">
                  <h4 className="truncate">{proxy.name}</h4>
                  <p className="id truncate text-xs font-mono">{proxy.proxyUrl}</p>
                </div>
                <div className="flex flex-col items-end gap-1 flex-shrink-0">
                  <span className="text-xs text-[var(--text-muted)]">Bound</span>
                  <strong className="text-lg">{proxy.boundConnectionCount || 0}</strong>
                </div>
              </div>

              <div className="flex flex-wrap gap-2 mb-4">
                <span className="admin-tag primary uppercase">{proxy.type || "http"}</span>
                {proxy.isActive ? (
                  <span className="admin-tag success">Active</span>
                ) : (
                  <span className="admin-tag danger">Disabled</span>
                )}
                {proxy.testStatus === "ok" && <span className="admin-tag success flex items-center gap-1"><CheckCircle2 size={12}/> OK</span>}
                {proxy.testStatus === "dead" && <span className="admin-tag danger flex items-center gap-1"><XCircle size={12}/> Dead</span>}
              </div>

              {proxy.lastError && (
                <div className="mb-4 text-xs text-[var(--danger)] bg-[var(--danger-light)] p-2 rounded truncate" title={proxy.lastError}>
                  {proxy.lastError}
                </div>
              )}

              <div className="flex justify-end gap-2 mt-4 pt-4 border-t border-[var(--border)]">
                <button
                  className="admin-btn admin-btn-sm admin-btn-ghost"
                  onClick={() => handleTest(proxy.id)}
                  title="Test connection"
                >
                  <Activity size={14} />
                  Test
                </button>
                <button
                  className="admin-btn admin-btn-sm admin-btn-ghost text-[var(--danger)] hover:bg-[var(--danger-light)]"
                  onClick={() => handleDelete(proxy.id)}
                  title="Delete"
                >
                  <Trash2 size={14} />
                  Delete
                </button>
              </div>
            </div>
          ))}
          {filtered.length === 0 && (
            <div className="col-span-full py-12 text-center text-[var(--text-muted)] border border-dashed border-[var(--border)] rounded-xl">
              <p>No proxies found.</p>
            </div>
          )}
        </div>
      </div>
      <ProxyAddModal 
        isOpen={addModalOpen} 
        onClose={() => setAddModalOpen(false)} 
        onAdd={handleAddSubmit}
        onAddBulk={handleBulkAdd}
      />
    </div>
  );
}
