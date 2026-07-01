"use client";

import { useState } from "react";
import { Input } from "@heroui/react";
import { X, Server, LayoutList } from "lucide-react";

export function ProxyAddModal({
  isOpen,
  onClose,
  onAdd,
  onAddBulk
}: {
  isOpen: boolean;
  onClose: () => void;
  onAdd: (name: string, url: string, type: string) => void;
  onAddBulk: (lines: string) => void;
}) {
  const [mode, setMode] = useState<"single" | "bulk">("single");
  const [name, setName] = useState("");
  const [url, setUrl] = useState("");
  const [type, setType] = useState("http");
  const [bulkText, setBulkText] = useState("");

  if (!isOpen) return null;

  const handleSubmit = () => {
    if (mode === "single") {
      if (!url) {
        alert("Proxy URL is required");
        return;
      }
      onAdd(name || "Custom Proxy", url, type);
    } else {
      if (!bulkText.trim()) {
        alert("Please enter proxies");
        return;
      }
      onAddBulk(bulkText);
    }
    onClose();
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50">
      <div className="bg-[var(--bg)] border border-[var(--border)] rounded-xl shadow-xl w-full max-w-md overflow-hidden">
        <div className="flex items-center justify-between p-4 border-b border-[var(--border)] bg-[var(--card-bg)]">
          <h3 className="font-semibold flex items-center gap-2">
            <Server size={18} />
            Add Proxy
          </h3>
          <button className="text-[var(--text-muted)] hover:text-[var(--text)] transition-colors" onClick={onClose}>
            <X size={18} />
          </button>
        </div>

        <div className="p-4">
          <div className="flex bg-[var(--card-bg)] rounded-lg p-1 mb-5 border border-[var(--border)]">
            <button
              className={`flex-1 py-1.5 text-xs font-medium rounded-md transition-colors ${
                mode === "single" ? "bg-[var(--primary)] text-white shadow" : "text-[var(--text-secondary)] hover:text-[var(--text)]"
              }`}
              onClick={() => setMode("single")}
            >
              Single
            </button>
            <button
              className={`flex-1 py-1.5 text-xs font-medium rounded-md transition-colors ${
                mode === "bulk" ? "bg-[var(--primary)] text-white shadow" : "text-[var(--text-secondary)] hover:text-[var(--text)]"
              }`}
              onClick={() => setMode("bulk")}
            >
              Bulk Add
            </button>
          </div>

          {mode === "single" ? (
            <div className="space-y-4">
              <div>
                <label className="text-xs font-semibold text-[var(--text-secondary)] block mb-1">Proxy Type</label>
                <select 
                  className="w-full bg-transparent border border-[var(--border)] rounded-lg px-3 py-2 text-sm text-[var(--text)] focus:outline-none focus:border-[var(--primary)]"
                  value={type}
                  onChange={(e) => setType(e.target.value)}
                >
                  <option value="http">HTTP / HTTPS</option>
                  <option value="socks5">SOCKS5</option>
                  <option value="vercel">Vercel Edge</option>
                </select>
              </div>
              <div>
                <label className="text-xs font-semibold text-[var(--text-secondary)] block mb-1">Proxy URL</label>
                <Input
                  placeholder="e.g. http://127.0.0.1:7890"
                  value={url}
                  onChange={(e) => setUrl(e.target.value)}
                  className="w-full"
                />
              </div>
              <div>
                <label className="text-xs font-semibold text-[var(--text-secondary)] block mb-1">Name (Optional)</label>
                <Input
                  placeholder="My SOCKS5 Proxy"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  className="w-full"
                />
              </div>
            </div>
          ) : (
            <div>
              <label className="text-xs font-semibold text-[var(--text-secondary)] block mb-1 flex justify-between">
                <span>Enter Proxies (One per line)</span>
                <span className="text-[var(--text-muted)] font-normal">Format: URL [Name]</span>
              </label>
              <textarea
                className="w-full h-40 bg-transparent border border-[var(--border)] rounded-lg p-3 text-sm font-mono text-[var(--text)] focus:outline-none focus:border-[var(--primary)] resize-none"
                placeholder={"http://127.0.0.1:7890 Local HTTP\nsocks5://user:pass@proxy.com:1080 Remote SOCKS"}
                value={bulkText}
                onChange={(e) => setBulkText(e.target.value)}
              />
            </div>
          )}
        </div>

        <div className="flex items-center justify-end gap-3 p-4 border-t border-[var(--border)] bg-[var(--card-bg)]">
          <button className="admin-btn admin-btn-ghost" onClick={onClose}>
            Cancel
          </button>
          <button className="admin-btn admin-btn-primary" onClick={handleSubmit}>
            {mode === "single" ? "Add Proxy" : "Bulk Add"}
          </button>
        </div>
      </div>
    </div>
  );
}
