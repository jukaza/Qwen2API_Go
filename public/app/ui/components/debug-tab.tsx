"use client";

import { useTranslation } from "react-i18next";
import { useCallback, useEffect, useRef, useState } from "react";
import {
  Bot,
  ChevronLeft,
  ChevronRight,
  ClipboardCopy,
  Eraser,
  Pencil,
  Send,
  Settings,
  Square,
  TerminalSquare,
} from "lucide-react";
import type { ModelItem } from "../types";

// ─── Types ────────────────────────────────────────────────────────────────────

type Role = "user" | "assistant";

interface ChatMessage {
  id: string;
  role: Role;
  content: string;
  tokens?: number;
  done: boolean;
}

interface TokenCount {
  prompt: number;
  completion: number;
}

const REASONING_OPTIONS = ["", "none", "minimal", "low", "medium", "high", "xhigh"] as const;

// ─── Helpers ──────────────────────────────────────────────────────────────────

function uid() {
  return Math.random().toString(36).slice(2, 10);
}

function parseSSEChunk(raw: string): string {
  let delta = "";
  const lines = raw.split("\n");
  for (const line of lines) {
    if (!line.startsWith("data:")) continue;
    const data = line.slice(5).trim();
    if (data === "[DONE]") break;
    try {
      const parsed = JSON.parse(data) as {
        choices?: Array<{ delta?: { content?: string }; finish_reason?: string | null }>;
        usage?: { prompt_tokens?: number; completion_tokens?: number };
      };
      const content = parsed.choices?.[0]?.delta?.content;
      if (typeof content === "string") delta += content;
    } catch {
      // skip malformed chunk
    }
  }
  return delta;
}

// ─── Sub-components ───────────────────────────────────────────────────────────

function CopyButton({ text }: { text: string }) {
  const [copied, setCopied] = useState(false);
  const copy = useCallback(async () => {
    try {
      await navigator.clipboard.writeText(text);
      setCopied(true);
      window.setTimeout(() => setCopied(false), 1500);
    } catch {
      // ignore
    }
  }, [text]);

  return (
    <button
      onClick={copy}
      title="Sao chép"
      className="chat-copy-btn"
      aria-label="Sao chép tin nhắn"
    >
      <ClipboardCopy size={13} />
      {copied ? <span>Đã sao chép</span> : null}
    </button>
  );
}

function UserBubble({ msg }: { msg: ChatMessage }) {
  return (
    <div className="chat-row chat-row-user">
      <div className="chat-bubble-user">
        <p className="chat-bubble-text">{msg.content}</p>
        <CopyButton text={msg.content} />
      </div>
    </div>
  );
}

function AssistantBubble({ msg }: { msg: ChatMessage }) {
  return (
    <div className="chat-row chat-row-assistant">
      <div className="chat-avatar">
        <Bot size={14} />
      </div>
      <div className="chat-bubble-assistant">
        <pre className="chat-bubble-text chat-pre">{msg.content || ""}{!msg.done ? <span className="chat-cursor">▋</span> : null}</pre>
        <div className="chat-bubble-footer">
          {msg.done && msg.tokens != null ? (
            <span className="chat-token-badge">{msg.tokens} tokens</span>
          ) : null}
          {msg.done ? <CopyButton text={msg.content} /> : null}
        </div>
      </div>
    </div>
  );
}

// ─── Main component ───────────────────────────────────────────────────────────

export function DebugTab({
  apiKey,
  models,
  defaultSystemPrompt,
}: {
  apiKey: string;
  models: ModelItem[];
  defaultSystemPrompt?: string;
}) {
  const { t } = useTranslation();
  const availableModels = models
    .map((m) => m.id)
    .filter((id) => {
      const lower = id.toLowerCase();
      return !lower.includes("image") && !lower.includes("video") && !lower.includes("flux");
    });


  // Settings state
  const [sidebarOpen, setSidebarOpen] = useState(true);
  const [model, setModel] = useState("");
  const [temperature, setTemperature] = useState(0.7);
  const [maxTokens, setMaxTokens] = useState(1024);
  const [reasoningEffort, setReasoningEffort] = useState<(typeof REASONING_OPTIONS)[number]>("");
  const [systemPrompt, setSystemPrompt] = useState(
    defaultSystemPrompt || "Bạn là trợ lý dùng để gỡ lỗi hệ thống, hãy trả lời trực tiếp và ngắn gọn.",
  );
  const [editingSystem, setEditingSystem] = useState(false);

  // Chat state
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [input, setInput] = useState("");
  const [streaming, setStreaming] = useState(false);
  const [error, setError] = useState("");
  const [tokenCount, setTokenCount] = useState<TokenCount>({ prompt: 0, completion: 0 });

  const abortRef = useRef<AbortController | null>(null);
  const bottomRef = useRef<HTMLDivElement | null>(null);
  const textareaRef = useRef<HTMLTextAreaElement | null>(null);

  const selectedModel = model || availableModels[0] || "";

  // Auto-scroll to bottom
  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages]);

  // Auto-resize textarea
  useEffect(() => {
    const el = textareaRef.current;
    if (!el) return;
    el.style.height = "auto";
    el.style.height = `${Math.min(el.scrollHeight, 160)}px`;
  }, [input]);

  const canSend = Boolean(selectedModel) && Boolean(input.trim()) && !streaming;

  async function sendMessage() {
    if (!canSend) return;
    const userText = input.trim();
    setInput("");
    setError("");

    const userMsg: ChatMessage = { id: uid(), role: "user", content: userText, done: true };
    const assistantId = uid();
    const assistantMsg: ChatMessage = { id: assistantId, role: "assistant", content: "", done: false };

    setMessages((prev) => [...prev, userMsg, assistantMsg]);
    setStreaming(true);

    const history = [...messages, userMsg];
    const apiMessages: Array<{ role: string; content: string }> = [];
    if (systemPrompt.trim()) apiMessages.push({ role: "system", content: systemPrompt.trim() });
    for (const m of history) apiMessages.push({ role: m.role, content: m.content });

    const body: Record<string, unknown> = {
      model: selectedModel,
      stream: true,
      temperature,
      max_tokens: maxTokens,
      messages: apiMessages,
      stream_options: { include_usage: true },
    };
    if (reasoningEffort) body.reasoning_effort = reasoningEffort;

    const ctrl = new AbortController();
    abortRef.current = ctrl;

    try {
      const resp = await fetch("/v1/chat/completions", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${apiKey}`,
        },
        body: JSON.stringify(body),
        signal: ctrl.signal,
        cache: "no-store",
      });

      if (!resp.ok || !resp.body) {
        const errText = await resp.text();
        throw new Error(errText || `HTTP ${resp.status}`);
      }

      const reader = resp.body.getReader();
      const decoder = new TextDecoder();
      let buffer = "";
      let accumulated = "";
      let promptTok = 0;
      let completionTok = 0;

      // eslint-disable-next-line no-constant-condition
      while (true) {
        const { value, done } = await reader.read();
        if (done) break;
        buffer += decoder.decode(value, { stream: true });

        // process complete SSE events (separated by double newlines)
        const parts = buffer.split("\n\n");
        buffer = parts.pop() ?? "";

        for (const part of parts) {
          if (!part.trim()) continue;
          // check for usage in final chunk
          if (part.includes('"usage"')) {
            try {
              const usageLine = part.split("\n").find((l) => l.startsWith("data:"));
              if (usageLine) {
                const parsed = JSON.parse(usageLine.slice(5).trim()) as {
                  usage?: { prompt_tokens?: number; completion_tokens?: number };
                };
                if (parsed.usage) {
                  promptTok = parsed.usage.prompt_tokens ?? promptTok;
                  completionTok = parsed.usage.completion_tokens ?? completionTok;
                }
              }
            } catch {
              // ignore parse errors
            }
          }
          const delta = parseSSEChunk(part);
          if (delta) {
            accumulated += delta;
            setMessages((prev) =>
              prev.map((m) =>
                m.id === assistantId ? { ...m, content: accumulated } : m,
              ),
            );
          }
        }
      }

      // finalize
      const finalTokens = completionTok || accumulated.split(" ").length;
      setMessages((prev) =>
        prev.map((m) =>
          m.id === assistantId ? { ...m, content: accumulated, done: true, tokens: finalTokens } : m,
        ),
      );
      setTokenCount((prev) => ({
        prompt: prev.prompt + promptTok,
        completion: prev.completion + finalTokens,
      }));
    } catch (err: unknown) {
      if (err instanceof Error && err.name === "AbortError") {
        setMessages((prev) =>
          prev.map((m) => (m.id === assistantId ? { ...m, done: true } : m)),
        );
      } else {
        const msg = err instanceof Error ? err.message : "Lỗi không xác định";
        setError(msg);
        setMessages((prev) => prev.filter((m) => m.id !== assistantId));
      }
    } finally {
      setStreaming(false);
      abortRef.current = null;
    }
  }

  function stopStreaming() {
    abortRef.current?.abort();
  }

  function clearChat() {
    setMessages([]);
    setTokenCount({ prompt: 0, completion: 0 });
    setError("");
  }

  function handleKeyDown(e: React.KeyboardEvent<HTMLTextAreaElement>) {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      void sendMessage();
    }
  }

  return (
    <div className="chat-playground">
      {/* ── Sidebar ─────────────────────────────────────────────────── */}
      <aside className={`chat-sidebar ${sidebarOpen ? "open" : "collapsed"}`}>
        <div className="chat-sidebar-header">
          {sidebarOpen ? (
            <>
              <Settings size={14} />
              <span>{t("debug.settings")}</span>
            </>
          ) : null}
          <button
            className="chat-sidebar-toggle"
            onClick={() => setSidebarOpen((v) => !v)}
            title={sidebarOpen ? "Thu gọn" : "Mở rộng"}
          >
            {sidebarOpen ? <ChevronLeft size={15} /> : <ChevronRight size={15} />}
          </button>
        </div>

        {sidebarOpen ? (
          <div className="chat-sidebar-body">
            {/* Model */}
            <div className="chat-setting-group">
              <label className="chat-setting-label">{t("debug.model")}</label>
              <select
                className="chat-select"
                value={selectedModel}
                onChange={(e) => setModel(e.target.value)}
              >
                {availableModels.map((id) => (
                  <option key={id} value={id}>{id}</option>
                ))}
              </select>
            </div>

            {/* Temperature */}
            <div className="chat-setting-group">
              <div className="chat-setting-row">
                <label className="chat-setting-label">{t("debug.temperature")}</label>
                <span className="chat-setting-value">{temperature.toFixed(1)}</span>
              </div>
              <input
                type="range"
                min={0}
                max={2}
                step={0.1}
                value={temperature}
                onChange={(e) => setTemperature(Number(e.target.value))}
                className="chat-slider"
              />
            </div>

            {/* Max Tokens */}
            <div className="chat-setting-group">
              <label className="chat-setting-label">{t("debug.maxTokens")}</label>
              <input
                type="number"
                className="chat-input-num"
                value={maxTokens}
                min={64}
                max={32768}
                step={256}
                onChange={(e) => setMaxTokens(Number(e.target.value))}
              />
            </div>

            {/* Reasoning Effort */}
            <div className="chat-setting-group">
              <label className="chat-setting-label">{t("debug.reasoningEffort")}</label>
              <select
                className="chat-select"
                value={reasoningEffort}
                onChange={(e) =>
                  setReasoningEffort(e.target.value as (typeof REASONING_OPTIONS)[number])
                }
              >
                <option value="">mặc định</option>
                {REASONING_OPTIONS.filter((o) => o).map((o) => (
                  <option key={o} value={o}>{o}</option>
                ))}
              </select>
            </div>

            {/* System Prompt */}
            <div className="chat-setting-group chat-setting-group-grow">
              <label className="chat-setting-label">{t("debug.systemPrompt")}</label>
              <textarea
                className="chat-textarea-system"
                rows={5}
                value={systemPrompt}
                onChange={(e) => setSystemPrompt(e.target.value)}
                placeholder="Nhập system prompt..."
              />
            </div>

            {/* Clear */}
            <button className="chat-clear-btn" onClick={clearChat} disabled={streaming}>
              <Eraser size={14} />
              {t("debug.clearResult")}
            </button>
          </div>
        ) : (
          /* collapsed icons */
          <div className="chat-sidebar-icons">
            <button title={t("debug.settings")} onClick={() => setSidebarOpen(true)}>
              <Settings size={16} />
            </button>
            <button title={t("debug.clearResult")} onClick={clearChat} disabled={streaming}>
              <Eraser size={16} />
            </button>
          </div>
        )}
      </aside>

      {/* ── Chat area ───────────────────────────────────────────────── */}
      <div className="chat-main">
        {/* Top bar */}
        <div className="chat-topbar">
          <div className="chat-topbar-left">
            <TerminalSquare size={14} />
            <span className="chat-model-badge">
              {selectedModel || "—"}
            </span>
            <span className={`chat-live-dot ${streaming ? "streaming" : ""}`} />
            {streaming ? <span className="chat-live-label">Đang stream...</span> : null}
          </div>
          <div className="chat-topbar-right">
            <span className="chat-token-counter">
              ↑ {tokenCount.prompt} &nbsp;/&nbsp; ↓ {tokenCount.completion} tokens
            </span>
          </div>
        </div>

        {/* Messages */}
        <div className="chat-messages">
          {messages.length === 0 ? (
            <div className="chat-empty">
              <Bot size={32} />
              <strong>{t("debug.emptyTitle")}</strong>
              <span>{t("debug.emptySubtitle")}</span>
            </div>
          ) : (
            messages.map((msg) =>
              msg.role === "user" ? (
                <UserBubble key={msg.id} msg={msg} />
              ) : (
                <AssistantBubble key={msg.id} msg={msg} />
              ),
            )
          )}
          {error ? (
            <div className="chat-error">
              <strong>Lỗi:</strong> {error}
            </div>
          ) : null}
          <div ref={bottomRef} />
        </div>

        {/* Input area */}
        <div className="chat-input-area">
          {/* System prompt strip */}
          <div className="chat-system-strip">
            {editingSystem ? (
              <div className="chat-system-edit">
                <textarea
                  className="chat-system-edit-textarea"
                  rows={3}
                  value={systemPrompt}
                  onChange={(e) => setSystemPrompt(e.target.value)}
                  placeholder="System prompt..."
                />
                <button
                  className="chat-system-edit-done"
                  onClick={() => setEditingSystem(false)}
                >
                  Xong
                </button>
              </div>
            ) : (
              <button
                className="chat-system-preview"
                onClick={() => setEditingSystem(true)}
                title="Chỉnh sửa System Prompt"
              >
                <span className="chat-system-label">System:</span>
                <span className="chat-system-text">
                  {systemPrompt.trim() || "Chưa đặt system prompt"}
                </span>
                <Pencil size={12} />
              </button>
            )}
          </div>

          {/* Input row */}
          <div className="chat-input-row">
            <textarea
              ref={textareaRef}
              className="chat-input-textarea"
              value={input}
              onChange={(e) => setInput(e.target.value)}
              onKeyDown={handleKeyDown}
              placeholder="Nhập tin nhắn... (Enter để gửi, Shift+Enter để xuống dòng)"
              rows={1}
              disabled={streaming && input === ""}
            />
            {streaming ? (
              <button className="chat-btn-stop" onClick={stopStreaming} title="Dừng">
                <Square size={16} />
              </button>
            ) : (
              <button
                className="chat-btn-send"
                onClick={() => void sendMessage()}
                disabled={!canSend}
                title="Gửi (Enter)"
              >
                <Send size={16} />
              </button>
            )}
          </div>
          <p className="chat-hint">Enter gửi · Shift+Enter xuống dòng · multi-turn</p>
        </div>
      </div>
    </div>
  );
}
