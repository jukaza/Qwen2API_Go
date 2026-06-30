"use client";

import {
  LayoutDashboard,
  Monitor,
  Users,
  Settings,
  Brain,
  Upload,
  Bug,
  ImageIcon,
  Video,
  MessageSquareText,
  Menu,
  Moon,
  Sun,
  RefreshCw,
  LogOut,
  ChevronLeft,
  ChevronRight,
  Globe,
  Sparkles,
} from "lucide-react";
import { Input } from "@heroui/react";
import { useTranslation } from "react-i18next";
import { useAdminConsole } from "./hooks/use-admin-console";
import { AccountsTab } from "./components/accounts-tab";
import { AssetGenerationTab } from "./components/asset-generation-tab";
import { DebugTab } from "./components/debug-tab";
import { ModelsTab } from "./components/models-tab";
import { OverviewTab } from "./components/overview-tab";
import { PromptsTab } from "./components/prompts-tab";
import { SettingsTab } from "./components/settings-tab";
import { UploadsTab } from "./components/uploads-tab";
import { DataScreenTab } from "./components/datascreen-tab";
import { formatCompactNumber } from "./components/dashboard-charts";
import { PROMPT_IDS, promptValue } from "./prompts";
import type { TabKey } from "./types";
import { useState, useEffect } from "react";

type MainTabKey = "monitoring" | "system" | "playground" | "devtools";

interface MainTabItem {
  key: MainTabKey;
  label: string;
  icon: React.ReactNode;
  subTabs: TabKey[];
  defaultTab: TabKey;
}

const LANG_OPTIONS = [
  { value: "vi", label: "langVi" },
  { value: "en", label: "langEn" },
];

export function AdminDashboard({ initialTab }: { initialTab?: TabKey } = {}) {
  const { t } = useTranslation();
  const { state, actions } = useAdminConsole(initialTab);
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);

  const MAIN_TABS: MainTabItem[] = [
    {
      key: "monitoring",
      label: t("nav.monitoring"),
      icon: <LayoutDashboard size={18} />,
      subTabs: ["overview", "datascreen"],
      defaultTab: "overview",
    },
    {
      key: "system",
      label: t("nav.system"),
      icon: <Settings size={18} />,
      subTabs: ["accounts", "settings", "models"],
      defaultTab: "accounts",
    },
    {
      key: "playground",
      label: t("nav.playground"),
      icon: <Sparkles size={18} />,
      subTabs: ["images", "videos", "uploads"],
      defaultTab: "images",
    },
    {
      key: "devtools",
      label: t("nav.devtools"),
      icon: <Bug size={18} />,
      subTabs: ["debug", "prompts"],
      defaultTab: "debug",
    },
  ];

  const SUB_TAB_LABELS: Record<TabKey, string> = {
    overview: t("nav.overview"),
    datascreen: t("nav.datascreen"),
    accounts: t("nav.accounts"),
    settings: t("nav.settings"),
    prompts: t("nav.prompts"),
    models: t("nav.models"),
    uploads: t("nav.uploads"),
    images: t("nav.images"),
    videos: t("nav.videos"),
    debug: t("nav.debug"),
  };

  const [renderedTabs, setRenderedTabs] = useState<Record<TabKey, boolean>>(() => {
    const initial = initialTab || "overview";
    return {
      overview: false,
      datascreen: false,
      accounts: false,
      settings: false,
      prompts: false,
      models: false,
      uploads: false,
      images: false,
      videos: false,
      debug: false,
      [initial]: true,
    } as Record<TabKey, boolean>;
  });

  useEffect(() => {
    setRenderedTabs((prev) => {
      if (prev[state.activeTab]) return prev;
      return { ...prev, [state.activeTab]: true };
    });
  }, [state.activeTab]);

  const currentMainTab = MAIN_TABS.find((item) => item.subTabs.includes(state.activeTab)) || MAIN_TABS[0];

  const [showKeyInput, setShowKeyInput] = useState(false);

  if (!state.verified) {
    if (!state.telegramConfigured) {
      return (
        <div className="admin-login">
          <div className="admin-login-card max-w-[450px]">
            <div className="flex items-center gap-3 mb-6">
              <div className="admin-sidebar-logo">Q2</div>
              <div>
                <h1>{t("appName")}</h1>
                <p className="text-[13px] text-[var(--text-secondary)] mt-1">{t("login.setupTelegramTitle")}</p>
              </div>
            </div>

            <p className="text-[13px] text-[var(--text-muted)] mb-4">{t("login.setupTelegramDesc")}</p>

            <div className="flex flex-col gap-4">
              <div>
                <label className="text-xs font-semibold text-[var(--text-secondary)] block mb-1">{t("login.botTokenLabel")}</label>
                <Input
                  placeholder={t("login.botTokenPlaceholder")}
                  type="text"
                  value={state.tgBotTokenInput}
                  onChange={(e) => actions.setTgBotTokenInput(e.target.value)}
                  className="w-full"
                />
              </div>

              <div>
                <label className="text-xs font-semibold text-[var(--text-secondary)] block mb-1">{t("login.adminChatIdLabel")}</label>
                <Input
                  placeholder={t("login.adminChatIdPlaceholder")}
                  type="text"
                  value={state.tgChatIdInput}
                  onChange={(e) => actions.setTgChatIdInput(e.target.value)}
                  className="w-full"
                />
                <span className="text-[11px] text-[var(--text-muted)] mt-1 block">{t("login.chatIdHint")}</span>
              </div>

              <div className="flex gap-3 mt-2">
                <button
                  className="admin-btn admin-btn-primary flex-1"
                  onClick={() => void actions.saveTelegramConfig()}
                  disabled={!state.tgBotTokenInput || !state.tgChatIdInput}
                >
                  {t("login.setupSave")}
                </button>
              </div>
            </div>

            {state.toast ? (
              <div
                className={`mt-4 p-3 rounded-lg text-sm font-medium ${
                  state.toast.type === "error"
                    ? "bg-[var(--danger-light)] text-[var(--danger)]"
                    : "bg-[var(--success-light)] text-[var(--success)]"
                }`}
              >
                {state.toast.message}
              </div>
            ) : null}
          </div>
        </div>
      );
    }

    if (state.loginStatus === "waiting") {
      return (
        <div className="admin-login">
          <div className="admin-login-card max-w-[420px] text-center">
            <div className="flex flex-col items-center justify-center gap-4 mb-6">
              <div className="relative flex items-center justify-center w-16 h-16 rounded-full bg-[var(--primary-light)] text-[var(--primary)] animate-pulse">
                <Brain size={32} />
              </div>
              <div>
                <h1 className="text-lg font-semibold">{t("login.waitingApproval")}</h1>
                <p className="text-[13px] text-[var(--text-secondary)] mt-2">{t("login.waitingApprovalDesc")}</p>
              </div>
            </div>

            <div className="p-4 bg-[var(--bg)] rounded-lg mb-6 text-left border border-[var(--border)] text-xs text-[var(--text-secondary)] flex flex-col gap-2">
              <div className="text-[var(--danger)] font-medium">
                {t("login.expiresIn", { secs: state.countdown })}
              </div>
            </div>

            <button
              className="admin-btn admin-btn-secondary w-full"
              onClick={() => actions.cancelTelegramLogin()}
            >
              {t("common.cancel")}
            </button>
          </div>
        </div>
      );
    }

    return (
      <div className="admin-login">
        <div className="admin-login-card">
          <div className="flex items-center gap-3 mb-6">
            <div className="admin-sidebar-logo">Q2</div>
            <div>
              <h1>{t("appName")}</h1>
              <p className="text-[13px] text-[var(--text-secondary)] mt-1">{t("login.title")}</p>
            </div>
          </div>

          <div className="flex flex-col gap-4">
            {!showKeyInput ? (
              <>
                <button
                  className="admin-btn admin-btn-primary w-full py-3"
                  onClick={() => void actions.startTelegramLogin()}
                >
                  <Sparkles size={16} />
                  {t("login.loginViaTelegram")}
                </button>

                <div className="relative flex items-center justify-center my-2">
                  <div className="absolute inset-0 flex items-center">
                    <div className="w-full border-t border-[var(--border)]"></div>
                  </div>
                  <span className="relative px-3 text-xs bg-[var(--card-bg)] text-[var(--text-muted)]">{t("login.or")}</span>
                </div>

                <button
                  className="text-xs font-medium text-[var(--primary)] hover:underline text-center"
                  onClick={() => setShowKeyInput(true)}
                >
                  {t("login.enter")}
                </button>
              </>
            ) : (
              <>
                <Input
                  placeholder={t("login.placeholder")}
                  type="password"
                  value={state.apiKeyInput}
                  onChange={(e) => actions.setApiKeyInput(e.target.value)}
                  className="w-full"
                  autoFocus
                />
                <div className="flex gap-3">
                  <button className="admin-btn admin-btn-primary flex-1" onClick={() => void actions.verifyAdmin()}>
                    {t("common.confirm")}
                  </button>
                  <button
                    className="admin-btn admin-btn-secondary"
                    onClick={() => {
                      actions.setApiKeyInput("");
                      if (typeof window !== "undefined") {
                        window.localStorage.removeItem("qwen2api-admin-key");
                      }
                    }}
                  >
                    {t("login.clear")}
                  </button>
                </div>

                <div className="relative flex items-center justify-center my-2">
                  <div className="absolute inset-0 flex items-center">
                    <div className="w-full border-t border-[var(--border)]"></div>
                  </div>
                  <span className="relative px-3 text-xs bg-[var(--card-bg)] text-[var(--text-muted)]">{t("login.or")}</span>
                </div>

                <button
                  className="text-xs font-medium text-[var(--primary)] hover:underline text-center"
                  onClick={() => setShowKeyInput(false)}
                >
                  {t("login.loginViaTelegram")}
                </button>
              </>
            )}
          </div>

          {(state.loginStatus === "rejected" || state.loginStatus === "expired" || state.toast) ? (
            <div
              className={`mt-4 p-3 rounded-lg text-sm font-medium ${
                state.loginStatus === "rejected" || state.loginStatus === "expired" || state.toast?.type === "error"
                  ? "bg-[var(--danger-light)] text-[var(--danger)]"
                  : state.toast?.type === "success"
                  ? "bg-[var(--success-light)] text-[var(--success)]"
                  : "bg-[var(--primary-light)] text-[var(--primary)]"
              }`}
            >
              {state.loginStatus === "rejected" && t("login.loginRejected")}
              {state.loginStatus === "expired" && t("login.loginExpired")}
              {!(state.loginStatus === "rejected" || state.loginStatus === "expired") && state.toast?.message}
            </div>
          ) : null}
        </div>
      </div>
    );
  }

  const currentTabLabel = SUB_TAB_LABELS[state.activeTab] || t("appName");

  return (
    <div className="admin-root">
      {state.toast ? (
        <div className={`admin-toast ${state.toast.type}`}>{state.toast.message}</div>
      ) : null}

      {/* Mobile overlay */}
      <div
        className={`admin-sidebar-overlay ${mobileMenuOpen ? "open" : ""}`}
        onClick={() => setMobileMenuOpen(false)}
      />

      {/* Sidebar */}
      <aside className={`admin-sidebar ${state.sidebarCollapsed ? "collapsed" : ""} ${mobileMenuOpen ? "mobile-open" : ""}`}>
        <div className="admin-sidebar-header">
          <div className="admin-sidebar-logo">Q2</div>
          <span className="admin-sidebar-title">{t("appName")}</span>
          <button
            className="admin-btn admin-btn-ghost admin-btn-sm ml-auto hidden lg:flex"
            onClick={actions.toggleSidebar}
            title={state.sidebarCollapsed ? "展开" : "收起"}
          >
            {state.sidebarCollapsed ? <ChevronRight size={16} /> : <ChevronLeft size={16} />}
          </button>
          <button
            className="admin-btn admin-btn-ghost admin-btn-sm ml-auto lg:hidden"
            onClick={() => setMobileMenuOpen(false)}
          >
            <ChevronLeft size={16} />
          </button>
        </div>

        <nav className="admin-sidebar-nav">
          {MAIN_TABS.map((item) => {
            const isActive = currentMainTab.key === item.key;
            return (
              <button
                key={item.key}
                type="button"
                className={`admin-nav-item ${isActive ? "active" : ""}`}
                onClick={() => {
                  actions.setActiveTab(item.defaultTab);
                  setMobileMenuOpen(false);
                }}
                title={item.label}
              >
                {item.icon}
                <span>{item.label}</span>
              </button>
            );
          })}
        </nav>

        <div className="admin-sidebar-footer">
          <button className="admin-nav-item" onClick={actions.toggleTheme}>
            {state.themeMode === "dark" ? <Sun size={18} /> : <Moon size={18} />}
            <span>{state.themeMode === "dark" ? t("common.themeLight") : t("common.themeDark")}</span>
          </button>
          <button className="admin-nav-item" onClick={() => void actions.refreshShell()}>
            <RefreshCw size={18} className={state.loadingShell ? "animate-spin" : ""} />
            <span>{state.loadingShell ? t("common.refreshing") : t("common.refresh")}</span>
          </button>
          <div className="admin-nav-item" style={{ cursor: "default" }}>
            <Globe size={18} />
            <select
              className="lang-select flex-1"
              value={state.language}
              onChange={(e) => actions.changeLanguage(e.target.value)}
            >
              {LANG_OPTIONS.map((opt) => (
                <option key={opt.value} value={opt.value}>
                  {t(`common.${opt.label}`)}
                </option>
              ))}
            </select>
          </div>
          <button className="admin-nav-item" onClick={actions.logout}>
            <LogOut size={18} />
            <span>{t("common.logout")}</span>
          </button>
        </div>
      </aside>

      {/* Main */}
      <div className={`admin-main ${state.sidebarCollapsed ? "collapsed" : ""}`}>
        {/* Header */}
        <header className="admin-header">
          <div className="admin-header-left">
            <button
              className="admin-btn admin-btn-ghost admin-btn-sm lg:hidden"
              onClick={() => setMobileMenuOpen(true)}
            >
              <Menu size={16} />
            </button>
            <span className="admin-page-title">{currentTabLabel}</span>
          </div>
          <div className="admin-header-right">
            <div className="hidden md:flex items-center gap-6 text-sm text-[var(--text-secondary)] mr-4">
              <div className="flex flex-col items-end">
                <span className="text-xs text-[var(--text-muted)]">{t("overview.totalRequests")}</span>
                <span className="font-semibold text-[var(--text)]">
                  {formatCompactNumber(state.overview?.analytics.totals.requests)}
                </span>
              </div>
              <div className="flex flex-col items-end">
                <span className="text-xs text-[var(--text-muted)]">{t("overview.accountValid")}</span>
                <span className="font-semibold text-[var(--text)]">
                  {formatCompactNumber(state.overview?.accounts.valid)}
                </span>
              </div>
              <div className="flex flex-col items-end">
                <span className="text-xs text-[var(--text-muted)]">{t("overview.modelTotal")}</span>
                <span className="font-semibold text-[var(--text)]">
                  {formatCompactNumber(state.modelCounts.total)}
                </span>
              </div>
            </div>
          </div>
        </header>

        {/* Content */}
        <main className="admin-content">
          {/* Sub-navigation ngang */}
          {currentMainTab.subTabs.length > 1 && (
            <div className="flex border-b border-[var(--border)] mb-6 overflow-x-auto gap-2 pb-2 scrollbar-none">
              {currentMainTab.subTabs.map((tabKey) => {
                const isActive = state.activeTab === tabKey;
                return (
                  <button
                    key={tabKey}
                    type="button"
                    onClick={() => actions.setActiveTab(tabKey)}
                    className={`px-4 py-1.5 text-xs font-semibold rounded-full transition-all duration-200 whitespace-nowrap ${
                      isActive
                        ? "bg-[var(--primary)] text-white shadow-sm"
                        : "text-[var(--text-secondary)] hover:bg-[var(--primary-light)] hover:text-[var(--primary)]"
                    }`}
                  >
                    {SUB_TAB_LABELS[tabKey]}
                  </button>
                );
              })}
            </div>
          )}

          {renderedTabs["overview"] && (
            <div style={{ display: state.activeTab === "overview" ? "block" : "none" }}>
              <OverviewTab overview={state.overview} modelCounts={state.modelCounts} />
            </div>
          )}

          {renderedTabs["datascreen"] && (
            <div style={{ display: state.activeTab === "datascreen" ? "block" : "none" }}>
              <DataScreenTab
                overview={state.overview}
                modelCounts={state.modelCounts}
                sseConnected={state.sseConnected}
              />
            </div>
          )}

          {renderedTabs["accounts"] && (
            <div style={{ display: state.activeTab === "accounts" ? "block" : "none" }}>
              <AccountsTab
                accounts={state.accounts}
                batchTask={state.batchTask}
                filters={state.filters}
                draftKeyword={state.draftKeyword}
                newAccountEmail={state.newAccountEmail}
                newAccountPassword={state.newAccountPassword}
                batchAccountsText={state.batchAccountsText}
                loadingAccounts={state.loadingAccounts}
                actions={{
                  setNewAccountEmail: actions.setNewAccountEmail,
                  setNewAccountPassword: actions.setNewAccountPassword,
                  setBatchAccountsText: actions.setBatchAccountsText,
                  createAccount: actions.createAccount,
                  createBatchTask: actions.createBatchTask,
                  refreshAccounts: actions.refreshAccounts,
                  setDraftKeyword: actions.setDraftKeyword,
                  setFilters: actions.setFilters,
                  refreshAccount: actions.refreshAccount,
                  deleteAccount: actions.deleteAccount,
                }}
              />
            </div>
          )}

          {renderedTabs["settings"] && (
            <div style={{ display: state.activeTab === "settings" ? "block" : "none" }}>
              <SettingsTab
                settings={state.settings}
                savingSettings={state.savingSettings}
                addKeyValue={state.addKeyValue}
                thresholdHours={state.thresholdHours}
                setAddKeyValue={actions.setAddKeyValue}
                setThresholdHours={actions.setThresholdHours}
                setSettings={actions.setSettings}
                addRegularKey={actions.addRegularKey}
                deleteRegularKey={actions.deleteRegularKey}
                refreshAllAccounts={actions.refreshAllAccounts}
                reloadRuntimeConfig={actions.reloadRuntimeConfig}
                saveSettings={actions.saveSettings}
                saveChatCleanupMode={actions.saveChatCleanupMode}
              />
            </div>
          )}

          {renderedTabs["prompts"] && (
            <div style={{ display: state.activeTab === "prompts" ? "block" : "none" }}>
              <PromptsTab
                prompts={state.prompts}
                savingSettings={state.savingSettings}
                savePrompts={actions.savePrompts}
                resetPrompts={actions.resetPrompts}
              />
            </div>
          )}

          {renderedTabs["models"] && (
            <div style={{ display: state.activeTab === "models" ? "block" : "none" }}>
              <ModelsTab
                models={state.filteredModels}
                keyword={state.modelKeyword}
                setKeyword={actions.setModelKeyword}
                refreshingModels={state.refreshingModels}
                refreshModels={actions.refreshModels}
              />
            </div>
          )}

          {renderedTabs["uploads"] && (
            <div style={{ display: state.activeTab === "uploads" ? "block" : "none" }}>
              <UploadsTab apiKey={state.apiKey} />
            </div>
          )}

          {renderedTabs["images"] && (
            <div style={{ display: state.activeTab === "images" ? "block" : "none" }}>
              <AssetGenerationTab
                kind="image"
                apiKey={state.apiKey}
                defaultPrompt={promptValue(state.prompts, PROMPT_IDS.imageDefault)}
              />
            </div>
          )}

          {renderedTabs["videos"] && (
            <div style={{ display: state.activeTab === "videos" ? "block" : "none" }}>
              <AssetGenerationTab
                kind="video"
                apiKey={state.apiKey}
                defaultPrompt={promptValue(state.prompts, PROMPT_IDS.videoDefault)}
              />
            </div>
          )}

          {renderedTabs["debug"] && (
            <div style={{ display: state.activeTab === "debug" ? "block" : "none" }}>
              <DebugTab
                apiKey={state.apiKey}
                models={state.filteredModels}
                defaultSystemPrompt={promptValue(state.prompts, PROMPT_IDS.debugSystem)}
              />
            </div>
          )}
        </main>
      </div>
    </div>
  );
}
