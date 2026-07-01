"use client";

import { useTranslation } from "react-i18next";
import { useState, useMemo } from "react";
import type { Dispatch, SetStateAction } from "react";
import type { SettingsResponse } from "../types";
import { Input, Switch } from "@heroui/react";
import {
  Save,
  Trash2,
  RefreshCw,
  RotateCcw,
  KeyRound,
  Settings2,
  SlidersHorizontal,
  Copy,
  Check,
  Shuffle,
  Search,
  Edit2,
  ShieldCheck,
  Shield,
  Plus
} from "lucide-react";

type SwitchValue = boolean | { target?: { checked?: boolean } };
type BooleanSettingKey = "autoRefresh" | "outThink" | "simpleModelMap";

function selectedSwitchValue(value: SwitchValue) {
  if (typeof value === "boolean") return value;
  return value.target?.checked ?? false;
}

export function SettingsTab({
  settings,
  savingSettings,
  addKeyValue,
  thresholdHours,
  setAddKeyValue,
  setThresholdHours,
  setSettings,
  addRegularKey,
  deleteRegularKey,
  updateAPIKey,
  refreshAllAccounts,
  reloadRuntimeConfig,
  saveSettings,
  saveChatCleanupMode,
}: {
  settings: SettingsResponse | null;
  savingSettings: boolean;
  addKeyValue: string;
  thresholdHours: string;
  setAddKeyValue: (value: string) => void;
  setThresholdHours: (value: string) => void;
  setSettings: Dispatch<SetStateAction<SettingsResponse | null>>;
  addRegularKey: (key: string, label: string, isAdmin: boolean) => Promise<void>;
  deleteRegularKey: (key: string) => Promise<void>;
  updateAPIKey: (key: string, label: string, isAdmin: boolean) => Promise<void>;
  refreshAllAccounts: (force: boolean) => Promise<void>;
  reloadRuntimeConfig: () => Promise<void>;
  saveSettings: (path: string, body: Record<string, unknown>, successMessage: string) => Promise<void>;
  saveChatCleanupMode: (mode: number) => Promise<void>;
}) {
  const { t } = useTranslation();

  const [newKey, setNewKey] = useState("");
  const [newLabel, setNewLabel] = useState("");
  const [newIsAdmin, setNewIsAdmin] = useState(false);
  const [keySearch, setKeySearch] = useState("");
  const [copiedKey, setCopiedKey] = useState<string | null>(null);

  // Editing state
  const [editingKey, setEditingKey] = useState<string | null>(null);
  const [editLabel, setEditLabel] = useState("");
  const [editIsAdmin, setEditIsAdmin] = useState(false);

  const generateRandomKey = () => {
    const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789";
    let result = "sk-";
    const randomArray = new Uint8Array(24);
    if (typeof window !== "undefined" && window.crypto) {
      window.crypto.getRandomValues(randomArray);
      for (let i = 0; i < 24; i++) {
        result += chars[randomArray[i] % chars.length];
      }
    } else {
      for (let i = 0; i < 24; i++) {
        result += chars.charAt(Math.floor(Math.random() * chars.length));
      }
    }
    setNewKey(result);
  };

  const copyToClipboard = (text: string) => {
    if (typeof navigator !== "undefined" && navigator.clipboard) {
      navigator.clipboard.writeText(text);
      setCopiedKey(text);
      setTimeout(() => setCopiedKey(null), 2000);
    }
  };

  const apiKeysList = useMemo(() => {
    if (settings?.apiKeys && settings.apiKeys.length > 0) {
      return settings.apiKeys;
    }
    const list = [];
    if (settings?.adminKey) {
      list.push({
        key: settings.adminKey,
        label: "Admin Key (Config)",
        isAdmin: true,
        createdAt: Math.floor(Date.now() / 1000),
      });
    }
    if (settings?.regularKeys) {
      for (const k of settings.regularKeys) {
        list.push({
          key: k,
          label: "Regular Key",
          isAdmin: false,
          createdAt: Math.floor(Date.now() / 1000),
        });
      }
    }
    return list;
  }, [settings]);

  const filteredKeys = useMemo(() => {
    const term = keySearch.toLowerCase().trim();
    if (!term) return apiKeysList;
    return apiKeysList.filter(
      (k) =>
        (k.label || "").toLowerCase().includes(term) ||
        (k.key || "").toLowerCase().includes(term)
    );
  }, [apiKeysList, keySearch]);

  const handleAddKey = async () => {
    const keyVal = newKey.trim();
    const labelVal = newLabel.trim() || "Regular Key";
    if (!keyVal) return;
    await addRegularKey(keyVal, labelVal, newIsAdmin);
    setNewKey("");
    setNewLabel("");
    setNewIsAdmin(false);
  };

  const handleUpdateKey = async (keyVal: string) => {
    const labelVal = editLabel.trim() || "Regular Key";
    await updateAPIKey(keyVal, labelVal, editIsAdmin);
    setEditingKey(null);
  };

  const startEdit = (key: string, label: string, isAdmin: boolean) => {
    setEditingKey(key);
    setEditLabel(label);
    setEditIsAdmin(isAdmin);
  };
  const enabledStrategies = [
    settings?.autoRefresh ?? false,
    settings?.outThink ?? false,
    settings?.simpleModelMap ?? false,
  ].filter(Boolean).length;

  const setBooleanSetting = (key: BooleanSettingKey, value: SwitchValue) => {
    const selected = selectedSwitchValue(value);
    setSettings((current) => (current ? { ...current, [key]: selected } : current));
  };

  return (
    <div className="flex flex-col gap-6">
      {/* Overview stats */}
      <div className="admin-stat-grid">
        <div className="admin-stat-card primary">
          <div className="label">{t("settings.enabledStrategies")}</div>
          <div className="value">{enabledStrategies}/3</div>
          <div className="desc">Tự động làm mới, xuất suy nghĩ (thinking), ánh xạ model</div>
        </div>
        <div className="admin-stat-card primary">
          <div className="label">{t("settings.regularKeyCount")}</div>
          <div className="value">{settings?.regularKeys.length ?? 0}</div>
          <div className="desc">API Key thông thường đã được đăng ký</div>
        </div>
        <div className="admin-stat-card primary">
          <div className="label">{t("settings.refreshInterval")}</div>
          <div className="value">{settings?.autoRefreshInterval ?? 21600}s</div>
          <div className="desc">Chu kỳ tự động làm mới token tài khoản</div>
        </div>
        <div className="admin-stat-card primary">
          <div className="label">{t("settings.searchMode")}</div>
          <div className="value">{settings?.searchInfoMode === "table" ? "Bảng" : "Văn bản"}</div>
          <div className="desc">Cách hiển thị mặc định của kết quả tìm kiếm</div>
        </div>
      </div>

      <div className="admin-settings-grid">
        <div className="flex flex-col gap-4">
          {/* Strategies */}
          <div className="admin-card">
            <div className="admin-card-header">
              <div>
                <h3><Settings2 size={16} className="inline mr-1" />{t("settings.strategies")}</h3>
                <p>Bật tắt chiến lược, tham số làm mới và cấu hình ánh xạ model</p>
              </div>
            </div>
            <div className="admin-card-body flex flex-col gap-6">
              <div>
                <h4 className="text-sm font-semibold mb-1">Cấu hình Bật/Tắt</h4>
                <p className="text-xs text-[var(--text-secondary)] mb-4">Quản lý các công tắc kích hoạt tính năng chạy nhanh</p>
                <div className="flex flex-col gap-3">
                  <SwitchCard
                    title={t("settings.autoRefresh")}
                    desc={t("settings.autoRefreshDesc")}
                    checked={settings?.autoRefresh ?? false}
                    onChange={(v) => setBooleanSetting("autoRefresh", v)}
                    onSave={() =>
                      settings &&
                      void saveSettings(
                        "/api/setAutoRefresh",
                        { autoRefresh: settings.autoRefresh, autoRefreshInterval: settings.autoRefreshInterval },
                        t("settings.saveAutoRefresh"),
                      )
                    }
                    saving={savingSettings}
                    disabled={!settings}
                    saveLabel={t("settings.saveAutoRefresh")}
                  />
                  <SwitchCard
                    title={t("settings.outThink")}
                    desc={t("settings.outThinkDesc")}
                    checked={settings?.outThink ?? false}
                    onChange={(v) => setBooleanSetting("outThink", v)}
                    onSave={() =>
                      settings &&
                      void saveSettings("/api/setOutThink", { outThink: settings.outThink }, t("settings.saveOutThink"))
                    }
                    saving={savingSettings}
                    disabled={!settings}
                    saveLabel={t("settings.saveOutThink")}
                    variant="ghost"
                  />
                  <SwitchCard
                    title={t("settings.simpleModelMap")}
                    desc={t("settings.simpleModelMapDesc")}
                    checked={settings?.simpleModelMap ?? false}
                    onChange={(v) => setBooleanSetting("simpleModelMap", v)}
                    onSave={() =>
                      settings &&
                      void saveSettings(
                        "/api/simple-model-map",
                        { simpleModelMap: settings.simpleModelMap },
                        t("settings.saveModelMap"),
                      )
                    }
                    saving={savingSettings}
                    disabled={!settings}
                    saveLabel={t("settings.saveModelMap")}
                    variant="secondary"
                  />
                </div>
              </div>

              <div>
                <h4 className="text-sm font-semibold mb-1">{t("settings.runParams")}</h4>
                <p className="text-xs text-[var(--text-secondary)] mb-4">Các tham số cấu hình chạy runtime</p>
                <div className="admin-form-grid">
                  <div className="admin-form-group">
                    <label>{t("settings.refreshIntervalLabel")}</label>
                    <Input
                      placeholder="Chu kỳ làm mới (giây)"
                      type="number"
                      value={String(settings?.autoRefreshInterval ?? 21600)}
                      onChange={(e) =>
                        setSettings((c) => (c ? { ...c, autoRefreshInterval: Number(e.target.value) || 0 } : c))
                      }
                    />
                    <button
                      className="admin-btn admin-btn-primary admin-btn-sm self-start mt-1"
                      disabled={!settings || savingSettings}
                      onClick={() =>
                        settings &&
                        void saveSettings(
                          "/api/setAutoRefresh",
                          { autoRefresh: settings.autoRefresh, autoRefreshInterval: settings.autoRefreshInterval },
                          t("settings.saveAutoRefresh"),
                        )
                      }
                    >
                      <Save size={14} />
                      {t("settings.saveRefreshParams")}
                    </button>
                  </div>
                  <div className="admin-form-group">
                    <label>{t("settings.batchConcurrency")}</label>
                    <Input
                      placeholder="Số luồng đồng thời"
                      type="number"
                      value={String(settings?.batchLoginConcurrency ?? 5)}
                      onChange={(e) =>
                        setSettings((c) => (c ? { ...c, batchLoginConcurrency: Number(e.target.value) || 1 } : c))
                      }
                    />
                    <button
                      className="admin-btn admin-btn-secondary admin-btn-sm self-start mt-1"
                      disabled={!settings || savingSettings}
                      onClick={() =>
                        settings &&
                        void saveSettings(
                          "/api/setBatchLoginConcurrency",
                          { batchLoginConcurrency: settings.batchLoginConcurrency },
                          t("settings.saveConcurrency"),
                        )
                      }
                    >
                      <Save size={14} />
                      {t("settings.saveConcurrency")}
                    </button>
                  </div>
                  <div className="admin-form-group">
                    <label>{t("settings.searchInfoMode")}</label>
                    <select
                      className="admin-select"
                      value={settings?.searchInfoMode ?? "text"}
                      onChange={(e) =>
                        setSettings((c) => (c ? { ...c, searchInfoMode: e.target.value as "table" | "text" } : c))
                      }
                    >
                      <option value="text">Văn bản</option>
                      <option value="table">Bảng</option>
                    </select>
                    <button
                      className="admin-btn admin-btn-secondary admin-btn-sm self-start mt-1"
                      disabled={!settings || savingSettings}
                      onClick={() =>
                        settings &&
                        void saveSettings(
                          "/api/search-info-mode",
                          { searchInfoMode: settings.searchInfoMode },
                          t("settings.saveSearchMode"),
                        )
                      }
                    >
                      <Save size={14} />
                      {t("settings.saveSearchMode")}
                    </button>
                  </div>
                  <div className="admin-form-group">
                    <label>{t("settings.chatCleanupMode")}</label>
                    <select
                      className="admin-select"
                      value={String(settings?.chatCleanupMode ?? 0)}
                      onChange={(e) =>
                        setSettings((c) => (c ? { ...c, chatCleanupMode: Number(e.target.value) } : c))
                      }
                    >
                      <option value="0">{t("settings.cleanupNone")}</option>
                      <option value="1">{t("settings.cleanupProgram")}</option>
                      <option value="2">{t("settings.cleanupAll")}</option>
                    </select>
                    <button
                      className="admin-btn admin-btn-secondary admin-btn-sm self-start mt-1"
                      disabled={!settings || savingSettings}
                      onClick={() => settings && void saveChatCleanupMode(settings.chatCleanupMode)}
                    >
                      <Save size={14} />
                      {t("settings.saveCleanupMode")}
                    </button>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>

        <div className="flex flex-col gap-4">
          {/* API Keys */}
          <div className="admin-card border border-[var(--border)] bg-[var(--card-bg)] shadow-md rounded-xl overflow-hidden">
            <div className="admin-card-header border-b border-[var(--border)] p-5 flex items-center justify-between">
              <div>
                <h3 className="text-base font-bold flex items-center gap-2 text-[var(--text-primary)]">
                  <KeyRound size={18} className="text-[var(--primary)]" />
                  {t("settings.apiKeys")}
                </h3>
                <p className="text-xs text-[var(--text-secondary)] mt-1">Quản lý và cấp quyền các khóa API truy cập hệ thống</p>
              </div>
            </div>
            <div className="admin-card-body p-5 flex flex-col gap-6">
              
              {/* Creation Form */}
              <div className="p-4 bg-[var(--bg-light)] border border-[var(--border)] rounded-xl flex flex-col gap-4">
                <h4 className="text-xs font-bold uppercase tracking-wider text-[var(--text-muted)]">Cấp Khóa API Mới</h4>
                
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  <div className="flex flex-col gap-1.5">
                    <label className="text-xs font-semibold text-[var(--text-secondary)]">Tên gợi nhớ / Label</label>
                    <Input
                      placeholder="Ví dụ: Dev Key, Bot Telegram..."
                      value={newLabel}
                      onChange={(e) => setNewLabel(e.target.value)}
                      className="w-full"
                    />
                  </div>
                  
                  <div className="flex flex-col gap-1.5">
                    <label className="text-xs font-semibold text-[var(--text-secondary)]">Giá trị API Key</label>
                    <div className="flex gap-2">
                      <Input
                        placeholder="Nhập hoặc tạo ngẫu nhiên..."
                        value={newKey}
                        onChange={(e) => setNewKey(e.target.value)}
                        className="flex-1"
                      />
                      <button
                        type="button"
                        onClick={generateRandomKey}
                        className="admin-btn admin-btn-secondary px-3"
                        title="Tạo khóa ngẫu nhiên"
                      >
                        <Shuffle size={15} />
                      </button>
                    </div>
                  </div>
                </div>

                <div className="flex flex-wrap items-center justify-between gap-4 mt-1 pt-3 border-t border-[var(--border)]">
                  <div className="flex items-center gap-2">
                    <span className="text-xs font-semibold text-[var(--text-secondary)]">Quyền Quản trị (Admin)</span>
                    <Switch
                      isSelected={newIsAdmin}
                      onChange={(checked) => setNewIsAdmin(selectedSwitchValue(checked))}
                    />
                  </div>
                  
                  <button
                    type="button"
                    className="admin-btn admin-btn-primary py-2.5 px-5 flex items-center gap-2"
                    onClick={handleAddKey}
                    disabled={!newKey.trim()}
                  >
                    <Plus size={15} />
                    Thêm Khóa mới
                  </button>
                </div>
              </div>

              {/* Search & List */}
              <div className="flex flex-col gap-4">
                <div className="flex items-center justify-between gap-4">
                  <h4 className="text-xs font-bold uppercase tracking-wider text-[var(--text-muted)]">
                    Danh Sách Khóa Hiện Tại ({filteredKeys.length})
                  </h4>
                  <div className="relative w-48 md:w-64">
                    <Search size={14} className="absolute left-3 top-1/2 -translate-y-1/2 text-[var(--text-muted)]" />
                    <input
                      type="text"
                      placeholder="Tìm kiếm khóa..."
                      value={keySearch}
                      onChange={(e) => setKeySearch(e.target.value)}
                      className="w-full bg-[var(--bg)] border border-[var(--border)] rounded-lg py-1.5 pl-9 pr-3 text-xs focus:outline-none focus:border-[var(--primary)] text-[var(--text-primary)] font-semibold"
                    />
                  </div>
                </div>

                <div className="grid grid-cols-1 gap-3 max-h-[400px] overflow-y-auto pr-1">
                  {filteredKeys.map((item) => {
                    const isEditing = editingKey === item.key;
                    const dateStr = item.createdAt 
                      ? new Date(item.createdAt * 1000).toLocaleString("vi-VN", { dateStyle: "short", timeStyle: "short" })
                      : "Không rõ";

                    return (
                      <div
                        key={item.key}
                        className={`p-4 rounded-xl border transition-all flex flex-col md:flex-row md:items-center justify-between gap-4 ${
                          item.isAdmin 
                            ? "bg-gradient-to-r from-[var(--warning-light)] to-[var(--bg-light)] border-[var(--warning)]/30 hover:border-[var(--warning)]/50" 
                            : "bg-[var(--bg-light)] border-[var(--border)] hover:border-[var(--text-muted)]"
                        }`}
                      >
                        <div className="flex flex-col gap-1.5 flex-1 min-w-0">
                          {isEditing ? (
                            <div className="flex flex-col gap-2 p-1">
                              <div className="flex gap-2 items-center">
                                <Input
                                  placeholder="Sửa tên..."
                                  value={editLabel}
                                  onChange={(e) => setEditLabel(e.target.value)}
                                  className="max-w-[200px]"
                                />
                                <div className="flex items-center gap-1">
                                  <span className="text-xs">Admin</span>
                                  <Switch
                                    size="sm"
                                    isSelected={editIsAdmin}
                                    onChange={(checked) => setEditIsAdmin(selectedSwitchValue(checked))}
                                  />
                                </div>
                              </div>
                              <div className="flex gap-2">
                                <button
                                  type="button"
                                  className="admin-btn admin-btn-primary admin-btn-sm text-xs py-1"
                                  onClick={() => handleUpdateKey(item.key)}
                                >
                                  Lưu
                                </button>
                                <button
                                  type="button"
                                  className="admin-btn admin-btn-secondary admin-btn-sm text-xs py-1"
                                  onClick={() => setEditingKey(null)}
                                >
                                  Hủy
                                </button>
                              </div>
                            </div>
                          ) : (
                            <div className="flex items-center gap-2 flex-wrap">
                              <span className="font-semibold text-sm text-[var(--text-primary)] truncate">
                                {item.label || "Regular Key"}
                              </span>
                              {item.isAdmin ? (
                                <span className="inline-flex items-center gap-1 text-[10px] font-bold px-2 py-0.5 rounded-full bg-[var(--danger-light)] text-[var(--danger)] border border-[var(--danger)]/20 animate-pulse">
                                  <ShieldCheck size={10} />
                                  Admin
                                </span>
                              ) : (
                                <span className="inline-flex items-center gap-1 text-[10px] font-bold px-2 py-0.5 rounded-full bg-[var(--primary-light)] text-[var(--primary)] border border-[var(--primary)]/20">
                                  <Shield size={10} />
                                  Regular
                                </span>
                              )}
                            </div>
                          )}

                          <div className="flex items-center gap-2">
                            <code className="text-xs bg-[var(--bg)] px-2.5 py-1 rounded border border-[var(--border)] text-[var(--text-secondary)] font-mono select-all">
                              {item.key}
                            </code>
                            <button
                              type="button"
                              onClick={() => copyToClipboard(item.key)}
                              className="p-1.5 hover:bg-[var(--border)] rounded text-[var(--text-muted)] hover:text-[var(--text-primary)] transition-colors"
                              title="Sao chép"
                            >
                              {copiedKey === item.key ? <Check size={14} className="text-[var(--success)]" /> : <Copy size={14} />}
                            </button>
                          </div>
                          
                          <span className="text-[10px] text-[var(--text-muted)]">
                            Ngày tạo: {dateStr}
                          </span>
                        </div>

                        <div className="flex items-center gap-2 self-end md:self-center">
                          {!isEditing && (
                            <button
                              type="button"
                              className="admin-btn admin-btn-secondary admin-btn-sm"
                              onClick={() => startEdit(item.key, item.label || "", item.isAdmin)}
                              title="Chỉnh sửa"
                            >
                              <Edit2 size={13} />
                            </button>
                          )}
                          <button
                            type="button"
                            className="admin-btn admin-btn-danger admin-btn-sm"
                            onClick={() => void deleteRegularKey(item.key)}
                            title="Xóa khóa"
                          >
                            <Trash2 size={13} />
                          </button>
                        </div>
                      </div>
                    );
                  })}

                  {filteredKeys.length === 0 && (
                    <div className="text-center py-8 border border-dashed border-[var(--border)] rounded-xl bg-[var(--bg-light)]">
                      <p className="text-sm text-[var(--text-muted)]">Không tìm thấy API Key nào phù hợp.</p>
                    </div>
                  )}
                </div>
              </div>

            </div>
          </div>

          {/* Refresh & Hot reload */}
          <div className="admin-card">
            <div className="admin-card-header">
              <div>
                <h3><SlidersHorizontal size={16} className="inline mr-1" />{t("settings.refreshAndReload")}</h3>
                <p>Làm mới tài khoản và tải lại tệp .env cấu hình</p>
              </div>
            </div>
            <div className="admin-card-body flex flex-col gap-5">
              <div className="admin-form-group">
                <label>{t("settings.thresholdHours")}</label>
                <Input
                  placeholder="Ngưỡng thời gian (giờ)"
                  type="number"
                  value={thresholdHours}
                  onChange={(e) => setThresholdHours(e.target.value)}
                />
              </div>

              <div className="flex gap-3">
                <button className="admin-btn admin-btn-secondary flex-1" onClick={() => void refreshAllAccounts(false)}>
                  <RefreshCw size={14} />
                  {t("settings.thresholdRefresh")}
                </button>
                <button className="admin-btn admin-btn-danger flex-1" onClick={() => void refreshAllAccounts(true)}>
                  <RotateCcw size={14} />
                  {t("settings.forceRefresh")}
                </button>
              </div>

              <div className="border-t border-[var(--border)] pt-4">
                <h4 className="text-sm font-semibold mb-1">{t("settings.hotReload")}</h4>
                <p className="text-xs text-[var(--text-secondary)] mb-3">
                  Nạp lại cấu hình .env sau khi sửa đổi thủ công
                </p>
                <button
                  className="admin-btn admin-btn-primary"
                  disabled={savingSettings}
                  onClick={() => void reloadRuntimeConfig()}
                >
                  <RotateCcw size={14} />
                  {t("settings.reloadEnv")}
                </button>
              </div>

              <div className="p-4 rounded-lg border border-[var(--danger)] bg-[var(--danger-light)] text-sm">
                <strong className="text-[var(--danger)] block mb-1">{t("settings.opsReminder")}</strong>
                <p className="text-[var(--text-secondary)]">{t("settings.opsReminderText")}</p>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

function SwitchCard({
  title,
  desc,
  checked,
  onChange,
  onSave,
  saving,
  disabled,
  saveLabel,
  variant = "primary",
}: {
  title: string;
  desc: string;
  checked: boolean;
  onChange: (v: SwitchValue) => void;
  onSave: () => void;
  saving: boolean;
  disabled: boolean;
  saveLabel: string;
  variant?: "primary" | "secondary" | "ghost";
}) {
  const variantClass =
    variant === "primary"
      ? "admin-btn-primary"
      : variant === "secondary"
      ? "admin-btn-secondary"
      : "admin-btn-ghost";

  return (
    <div className="admin-switch-card">
      <div>
        <strong>{title}</strong>
        <p>{desc}</p>
      </div>
      <div className="flex flex-col items-end gap-3">
        <Switch isSelected={checked} onChange={(value) => onChange(value)}>
          <Switch.Control>
            <Switch.Thumb />
          </Switch.Control>
        </Switch>
        <button className={`admin-btn ${variantClass} admin-btn-sm`} disabled={disabled || saving} onClick={onSave}>
          <Save size={14} />
          {saveLabel}
        </button>
      </div>
    </div>
  );
}

function PlusIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
      <line x1="12" y1="5" x2="12" y2="19" />
      <line x1="5" y1="12" x2="19" y2="12" />
    </svg>
  );
}
