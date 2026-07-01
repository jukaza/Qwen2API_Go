package main

import (
	"fmt"
	"log"

	"qwen2api/internal/config"
	"qwen2api/internal/storage"
)

func main() {
	log.Println("Bắt đầu di chuyển dữ liệu từ File JSON sang SQLite...")

	// Khởi tạo stores cho File mode
	cfgFile := config.Config{DataSaveMode: "file"}
	fileAccountStore, err := storage.NewAccountStore(cfgFile)
	if err != nil {
		log.Fatalf("Lỗi khởi tạo File AccountStore: %v", err)
	}
	fileProxyStore, err := storage.NewProxyStore(cfgFile)
	if err != nil {
		log.Fatalf("Lỗi khởi tạo File ProxyStore: %v", err)
	}
	fileConvStore, err := storage.NewConversationStore(cfgFile)
	if err != nil {
		log.Fatalf("Lỗi khởi tạo File ConversationStore: %v", err)
	}
	fileSessionStore, err := storage.NewSessionStore(cfgFile)
	if err != nil {
		log.Fatalf("Lỗi khởi tạo File SessionStore: %v", err)
	}

	// Khởi tạo stores cho SQLite mode
	cfgSQLite := config.Config{DataSaveMode: "sqlite", SQLiteDBPath: "data/data.db"}
	sqliteAccountStore, err := storage.NewAccountStore(cfgSQLite)
	if err != nil {
		log.Fatalf("Lỗi khởi tạo SQLite AccountStore: %v", err)
	}
	sqliteProxyStore, err := storage.NewProxyStore(cfgSQLite)
	if err != nil {
		log.Fatalf("Lỗi khởi tạo SQLite ProxyStore: %v", err)
	}
	sqliteConvStore, err := storage.NewConversationStore(cfgSQLite)
	if err != nil {
		log.Fatalf("Lỗi khởi tạo SQLite ConversationStore: %v", err)
	}
	sqliteSessionStore, err := storage.NewSessionStore(cfgSQLite)
	if err != nil {
		log.Fatalf("Lỗi khởi tạo SQLite SessionStore: %v", err)
	}

	// Migrate Accounts
	accs, err := fileAccountStore.LoadAccounts()
	if err != nil {
		log.Printf("Cảnh báo: Lỗi khi load Accounts từ file: %v", err)
	} else if len(accs) > 0 {
		if err := sqliteAccountStore.SaveAllAccounts(accs); err != nil {
			log.Fatalf("Lỗi lưu Accounts vào SQLite: %v", err)
		}
		log.Printf("Đã chuyển %d accounts.", len(accs))
	}

	// Migrate Proxies
	proxies, err := fileProxyStore.LoadProxies()
	if err != nil {
		log.Printf("Cảnh báo: Lỗi khi load Proxies từ file: %v", err)
	} else if len(proxies) > 0 {
		if err := sqliteProxyStore.SaveAllProxies(proxies); err != nil {
			log.Fatalf("Lỗi lưu Proxies vào SQLite: %v", err)
		}
		log.Printf("Đã chuyển %d proxies.", len(proxies))
	}

	// Migrate Conversations
	convs, err := fileConvStore.ListConversationSessions()
	if err != nil {
		log.Printf("Cảnh báo: Lỗi khi load Conversations từ file: %v", err)
	} else if len(convs) > 0 {
		for _, c := range convs {
			_ = sqliteConvStore.SaveConversationSession(c)
		}
		log.Printf("Đã chuyển %d conversation sessions.", len(convs))
	}

	// Migrate Sessions
	sessions, err := fileSessionStore.ListSessions()
	if err != nil {
		log.Printf("Cảnh báo: Lỗi khi load Sessions từ file: %v", err)
	} else if len(sessions) > 0 {
		for _, s := range sessions {
			_ = sqliteSessionStore.SaveSession(s)
		}
		log.Printf("Đã chuyển %d web sessions.", len(sessions))
	}

	fmt.Println("Hoàn tất di chuyển dữ liệu sang SQLite thành công!")
}
