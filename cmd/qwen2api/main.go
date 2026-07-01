package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"qwen2api/internal/account"
	"qwen2api/internal/admin"
	"qwen2api/internal/auth"
	"qwen2api/internal/cleanup"
	"qwen2api/internal/config"
	"qwen2api/internal/logging"
	"qwen2api/internal/metrics"
	"qwen2api/internal/openai"
	"qwen2api/internal/proxy"
	"qwen2api/internal/qwen"
	"qwen2api/internal/server"
	"qwen2api/internal/storage"
	"qwen2api/internal/telegram"
)

func main() {
	if err := config.EnsureDotEnv(config.DefaultEnvPath); err != nil {
		panic(err)
	}
	config.LoadDotEnv(config.DefaultEnvPath)
	cfg := config.Load()
	logger := logging.New(cfg.DebugMode)

	if len(cfg.APIKeys) == 0 {
		logger.ErrorModule("APP", "请务必设置 API_KEY 环境变量")
		os.Exit(1)
	}

	store, err := storage.NewAccountStore(cfg)
	if err != nil {
		logger.ErrorModule("APP", "初始化账号存储失败: %v", err)
		os.Exit(1)
	}
	conversationStore, err := storage.NewConversationStore(cfg)
	if err != nil {
		logger.ErrorModule("APP", "初始化会话存储失败: %v", err)
		os.Exit(1)
	}
	sessionStore, err := storage.NewSessionStore(cfg)
	if err != nil {
		logger.ErrorModule("APP", "初始化Session存储失败: %v", err)
		os.Exit(1)
	}
	proxyStore, err := storage.NewProxyStore(cfg)
	if err != nil {
		logger.ErrorModule("APP", "初始化Proxy存储失败: %v", err)
		os.Exit(1)
	}

	apiKeyStore, err := storage.NewAPIKeyStore(cfg)
	if err != nil {
		logger.ErrorModule("APP", "初始化API Key存储失败: %v", err)
		os.Exit(1)
	}

	proxyMgr := proxy.NewManager(proxyStore, store, logger)

	keyring := auth.NewKeyring(cfg.APIKeys, cfg.AdminKey)

	// Load keys from store. If store is empty, migrate default keys from config.
	loadedKeys, err := apiKeyStore.LoadAPIKeys()
	if err == nil && len(loadedKeys) == 0 {
		var keysToSave []storage.APIKey
		for _, k := range cfg.APIKeys {
			keysToSave = append(keysToSave, storage.APIKey{
				Key:       k,
				Label:     "Default Key",
				IsAdmin:   k == cfg.AdminKey,
				CreatedAt: time.Now().Unix(),
			})
		}
		if len(keysToSave) == 0 {
			keysToSave = append(keysToSave, storage.APIKey{
				Key:       "sk-user-change-me",
				Label:     "Default Admin Key",
				IsAdmin:   true,
				CreatedAt: time.Now().Unix(),
			})
		}
		if err := apiKeyStore.SaveAllAPIKeys(keysToSave); err != nil {
			logger.WarnModule("APP", "初始化默认 API Key 失败: %v", err)
		} else {
			logger.InfoModule("APP", "已将默认 API Keys 导入存储中")
			loadedKeys = keysToSave
		}
	}
	keyring.SyncFromStore(loadedKeys)

	runtime := config.NewRuntime(cfg)
	stats := metrics.NewDashboardStats()
	qwenClient := qwen.NewClient(cfg, logger)
	accountService := account.NewService(cfg, runtime, store, qwenClient, proxyMgr, logger)
	conversationSessions := openai.NewConversationSessionService(conversationStore, logger)
	chatTracker, err := storage.NewChatTracker(cfg)
	if err != nil {
		logger.WarnModule("APP", "初始化对话追踪器失败: %v", err)
		chatTracker = nil
	}

	cleanupService := cleanup.NewService(cfg, runtime, accountService, qwenClient, chatTracker, logger)
	cleanupService.Start()

	tgService := telegram.NewService(cfg.TelegramBotToken, cfg.TelegramAdminChatID, sessionStore, logger)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	tgService.Start(ctx)
	defer tgService.Stop()

	defer accountService.Close()
	defer cleanupService.Stop()

	openAIHandler := openai.NewHandler(cfg, runtime, qwenClient, accountService, conversationSessions, proxyMgr, chatTracker, stats, logger)
	adminHandler := admin.NewHandler(cfg, runtime, keyring, accountService, openAIHandler, stats, logger, sessionStore, tgService, proxyStore, proxyMgr, apiKeyStore)
	httpServer := server.New(cfg, keyring, openAIHandler, adminHandler, stats, logger, sessionStore)
	serverErrCh := make(chan error, 1)

	go func() {
		logger.InfoModule("APP", "服务器启动中，监听 %s:%d", cfg.ListenAddressOrDefault(), cfg.ListenPort)
		serverErrCh <- httpServer.ListenAndServe()
	}()

	go func() {
		logger.InfoModule("ACCOUNT", "账号池后台初始化开始")
		if initErr := accountService.Initialize(context.Background()); initErr != nil {
			logger.ErrorModule("ACCOUNT", "账号池后台初始化失败: %v", initErr)
			return
		}
		logger.InfoModule("ACCOUNT", "账号池后台初始化完成")
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = httpServer.Shutdown(shutdownCtx)
	case err = <-serverErrCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.ErrorModule("APP", "服务器启动失败: %v", err)
			os.Exit(1)
		}
	}
}
