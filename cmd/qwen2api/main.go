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
		logger.ErrorModule("APP", "Vui lòng thiết lập biến môi trường API_KEY")
		os.Exit(1)
	}

	store, err := storage.NewAccountStore(cfg)
	if err != nil {
		logger.ErrorModule("APP", "Khởi tạo lưu trữ tài khoản thất bại: %v", err)
		os.Exit(1)
	}
	conversationStore, err := storage.NewConversationStore(cfg)
	if err != nil {
		logger.ErrorModule("APP", "Khởi tạo lưu trữ hội thoại thất bại: %v", err)
		os.Exit(1)
	}
	sessionStore, err := storage.NewSessionStore(cfg)
	if err != nil {
		logger.ErrorModule("APP", "Khởi tạo lưu trữ session thất bại: %v", err)
		os.Exit(1)
	}
	proxyStore, err := storage.NewProxyStore(cfg)
	if err != nil {
		logger.ErrorModule("APP", "Khởi tạo lưu trữ proxy thất bại: %v", err)
		os.Exit(1)
	}

	apiKeyStore, err := storage.NewAPIKeyStore(cfg)
	if err != nil {
		logger.ErrorModule("APP", "Khởi tạo lưu trữ API Key thất bại: %v", err)
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
			logger.WarnModule("APP", "Khởi tạo API Key mặc định thất bại: %v", err)
		} else {
			logger.InfoModule("APP", "Đã nhập API Keys mặc định vào bộ nhớ lưu trữ")
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
		logger.WarnModule("APP", "Khởi tạo trình theo dõi hội thoại thất bại: %v", err)
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
		logger.InfoModule("APP", "Máy chủ đang khởi động, lắng nghe tại %s:%d", cfg.ListenAddressOrDefault(), cfg.ListenPort)
		serverErrCh <- httpServer.ListenAndServe()
	}()

	go func() {
		logger.InfoModule("ACCOUNT", "Bắt đầu khởi tạo ngầm bể tài khoản")
		if initErr := accountService.Initialize(context.Background()); initErr != nil {
			logger.ErrorModule("ACCOUNT", "Khởi tạo ngầm bể tài khoản thất bại: %v", initErr)
			return
		}
		logger.InfoModule("ACCOUNT", "Khởi tạo ngầm bể tài khoản hoàn tất")
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = httpServer.Shutdown(shutdownCtx)
	case err = <-serverErrCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.ErrorModule("APP", "Máy chủ khởi động thất bại: %v", err)
			os.Exit(1)
		}
	}
}
