package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"qwen2api/internal/logging"
	"qwen2api/internal/storage"
)

type LoginRequest struct {
	ID        string    `json:"id"`
	IP        string    `json:"ip"`
	UserAgent string    `json:"user_agent"`
	CreatedAt time.Time `json:"created_at"`
	Status    string    `json:"status"` // "pending", "approved", "rejected", "expired"
	Token     string    `json:"token,omitempty"`
	MsgID     int       `json:"msg_id,omitempty"`
}

type Service struct {
	mu           sync.RWMutex
	token        string
	adminChat    string
	logger       *logging.Logger
	client       *http.Client
	sessions     storage.SessionStore
	requests     *sync.Map // maps requestID (string) -> *LoginRequest
	cancelPoller context.CancelFunc
}

func NewService(token, adminChat string, sessions storage.SessionStore, logger *logging.Logger) *Service {
	return &Service{
		token:     strings.TrimSpace(token),
		adminChat: strings.TrimSpace(adminChat),
		logger:    logger,
		client:    &http.Client{Timeout: 35 * time.Second},
		sessions:  sessions,
		requests:  &sync.Map{},
	}
}

func (s *Service) Start(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cancelPoller != nil {
		s.cancelPoller()
	}

	if s.token == "" {
		s.logger.InfoModule("TELEGRAM", "Telegram token is empty. Bot is disabled.")
		return
	}

	pollCtx, cancel := context.WithCancel(ctx)
	s.cancelPoller = cancel

	go s.pollLoop(pollCtx)
	go s.janitorLoop(pollCtx)
	s.logger.InfoModule("TELEGRAM", "Telegram service started.")
}

func (s *Service) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancelPoller != nil {
		s.cancelPoller()
		s.cancelPoller = nil
	}
	s.logger.InfoModule("TELEGRAM", "Telegram service stopped.")
}

func (s *Service) Reload(token, adminChat string) {
	s.mu.Lock()
	s.token = strings.TrimSpace(token)
	s.adminChat = strings.TrimSpace(adminChat)
	s.mu.Unlock()

	s.logger.InfoModule("TELEGRAM", "Reloading Telegram configuration. Restarting poller...")
	// Restart poller with current parent context
	s.Start(context.Background())
}

func (s *Service) GetRequestsMap() *sync.Map {
	return s.requests
}

func (s *Service) getTokenAndChat() (string, string) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.token, s.adminChat
}

func (s *Service) callTelegram(method string, params map[string]any) ([]byte, error) {
	token, _ := s.getTokenAndChat()
	if token == "" {
		return nil, fmt.Errorf("bot token is empty")
	}

	urlStr := fmt.Sprintf("https://api.telegram.org/bot%s/%s", token, method)
	body, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", urlStr, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return respBody, fmt.Errorf("telegram returned status %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

func (s *Service) SendLoginRequest(req *LoginRequest) error {
	_, adminChat := s.getTokenAndChat()
	if adminChat == "" {
		return fmt.Errorf("admin chat ID is not configured")
	}

	text := fmt.Sprintf(
		"🔔 <b>Yêu cầu đăng nhập mới</b>\n\n"+
			"• <b>IP:</b> <code>%s</code>\n"+
			"• <b>Thiết bị:</b> %s\n"+
			"• <b>Thời gian:</b> %s\n\n"+
			"⚠️ Vui lòng phê duyệt hoặc từ chối yêu cầu đăng nhập này trong vòng 2 phút.",
		htmlEscape(req.IP), htmlEscape(req.UserAgent), htmlEscape(req.CreatedAt.Local().Format("2006-01-02 15:04:05")),
	)

	keyboard := map[string]any{
		"inline_keyboard": [][]map[string]any{
			{
				{"text": "✅ Phê duyệt / Approve", "callback_data": "login:approve:" + req.ID},
				{"text": "❌ Từ chối / Reject", "callback_data": "login:reject:" + req.ID},
			},
		},
	}

	params := map[string]any{
		"chat_id":      adminChat,
		"text":         text,
		"parse_mode":   "HTML",
		"reply_markup": keyboard,
	}

	respBytes, err := s.callTelegram("sendMessage", params)
	if err != nil {
		s.logger.ErrorModule("TELEGRAM", "Failed to send login request message: %v", err)
		return err
	}

	var result struct {
		Ok     bool `json:"ok"`
		Result struct {
			MessageID int `json:"message_id"`
		} `json:"result"`
	}

	if err := json.Unmarshal(respBytes, &result); err == nil && result.Ok {
		req.MsgID = result.Result.MessageID
	}

	return nil
}

func (s *Service) updateMessageStatus(req *LoginRequest, newText string) {
	_, adminChat := s.getTokenAndChat()
	if adminChat == "" || req.MsgID == 0 {
		return
	}

	params := map[string]any{
		"chat_id":    adminChat,
		"message_id": req.MsgID,
		"text":       newText,
		"parse_mode": "HTML",
	}

	_, _ = s.callTelegram("editMessageText", params)
}

func (s *Service) pollLoop(ctx context.Context) {
	offset := 0
	s.logger.InfoModule("TELEGRAM", "Telegram update poller loop started.")

	for {
		select {
		case <-ctx.Done():
			s.logger.InfoModule("TELEGRAM", "Telegram update poller loop stopped.")
			return
		default:
		}

		token, _ := s.getTokenAndChat()
		if token == "" {
			time.Sleep(2 * time.Second)
			continue
		}

		params := map[string]any{
			"offset":  offset,
			"timeout": 30,
		}

		respBytes, err := s.callTelegram("getUpdates", params)
		if err != nil {
			s.logger.WarnModule("TELEGRAM", "getUpdates failed: %v. Retrying in 5 seconds...", err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(5 * time.Second):
			}
			continue
		}

		var updateResp tgUpdatesResponse
		if err := json.Unmarshal(respBytes, &updateResp); err != nil {
			s.logger.ErrorModule("TELEGRAM", "Failed to parse updates: %v", err)
			time.Sleep(2 * time.Second)
			continue
		}

		if !updateResp.Ok {
			s.logger.ErrorModule("TELEGRAM", "Telegram updates response is not ok")
			time.Sleep(2 * time.Second)
			continue
		}

		for _, update := range updateResp.Result {
			if update.UpdateID >= offset {
				offset = update.UpdateID + 1
			}

			s.handleUpdate(update)
		}
	}
}

func (s *Service) handleUpdate(update tgUpdate) {
	if update.Message != nil {
		s.handleMessage(update.Message)
	} else if update.CallbackQuery != nil {
		s.handleCallbackQuery(update.CallbackQuery)
	}
}

func (s *Service) handleMessage(msg *tgMessage) {
	if msg == nil || msg.Chat == nil {
		return
	}

	text := strings.TrimSpace(msg.Text)
	if text == "/start" {
		chatIdStr := strconv.FormatInt(msg.Chat.ID, 10)
		reply := fmt.Sprintf(
			"👋 <b>Chào bạn!</b>\n\n"+
				"• User ID/Chat ID của bạn là: <code>%s</code>\n\n"+
				"Hãy copy ID này để điền vào phần cấu hình trang quản trị Dashboard.",
			htmlEscape(chatIdStr),
		)
		params := map[string]any{
			"chat_id":    msg.Chat.ID,
			"text":       reply,
			"parse_mode": "HTML",
		}
		_, _ = s.callTelegram("sendMessage", params)
	}
}

func (s *Service) handleCallbackQuery(cb *tgCallbackQuery) {
	if cb == nil {
		return
	}

	// Answer callback query first
	defer func() {
		params := map[string]any{
			"callback_query_id": cb.ID,
		}
		_, _ = s.callTelegram("answerCallbackQuery", params)
	}()

	_, adminChat := s.getTokenAndChat()
	senderID := strconv.FormatInt(cb.From.ID, 10)

	if adminChat != "" && senderID != adminChat {
		s.logger.WarnModule("TELEGRAM", "Callback query from unauthorized user: %s (expected %s)", senderID, adminChat)
		// send unauthorized warning
		params := map[string]any{
			"callback_query_id": cb.ID,
			"text":              "⚠️ Bạn không có quyền phê duyệt yêu cầu này!",
			"show_alert":        true,
		}
		_, _ = s.callTelegram("answerCallbackQuery", params)
		return
	}

	data := cb.Data
	if !strings.HasPrefix(data, "login:") {
		return
	}

	parts := strings.Split(data, ":")
	if len(parts) < 3 {
		return
	}

	action := parts[1] // "approve" or "reject"
	requestID := parts[2]

	val, ok := s.requests.Load(requestID)
	if !ok {
		// Yêu cầu không tồn tại hoặc đã hết hạn
		s.logger.WarnModule("TELEGRAM", "Login request ID not found or expired: %s", requestID)
		if cb.Message != nil && cb.Message.Chat != nil {
			params := map[string]any{
				"chat_id":    cb.Message.Chat.ID,
				"message_id": cb.Message.MessageID,
				"text":       "⏳ <b>Yêu cầu đăng nhập đã hết hạn hoặc không tồn tại.</b>",
				"parse_mode": "HTML",
			}
			_, _ = s.callTelegram("editMessageText", params)
		}
		return
	}

	req := val.(*LoginRequest)
	if req.Status != "pending" {
		return // Already processed
	}

	if action == "approve" {
		req.Status = "approved"
		s.logger.InfoModule("TELEGRAM", "Login request approved: requestId=%s ip=%s", requestID, req.IP)

		// Edit Telegram Message
		newText := fmt.Sprintf(
			"✅ <b>Yêu cầu đăng nhập ĐÃ ĐƯỢC PHÊ DUYỆT</b>\n\n"+
				"• <b>IP:</b> <code>%s</code>\n"+
				"• <b>Thiết bị:</b> %s\n"+
				"• <b>Thời gian:</b> %s\n\n"+
				"🚪 Trình duyệt đã được phép đăng nhập trang quản trị.",
			htmlEscape(req.IP), htmlEscape(req.UserAgent), htmlEscape(req.CreatedAt.Local().Format("2006-01-02 15:04:05")),
		)
		s.updateMessageStatus(req, newText)

	} else if action == "reject" {
		req.Status = "rejected"
		s.logger.InfoModule("TELEGRAM", "Login request rejected: requestId=%s ip=%s", requestID, req.IP)

		// Edit Telegram Message
		newText := fmt.Sprintf(
			"❌ <b>Yêu cầu đăng nhập ĐÃ BỊ TỪ CHỐI</b>\n\n"+
				"• <b>IP:</b> <code>%s</code>\n"+
				"• <b>Thiết bị:</b> %s\n"+
				"• <b>Thời gian:</b> %s",
			htmlEscape(req.IP), htmlEscape(req.UserAgent), htmlEscape(req.CreatedAt.Local().Format("2006-01-02 15:04:05")),
		)
		s.updateMessageStatus(req, newText)
	}
}

func (s *Service) janitorLoop(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			now := time.Now()
			s.requests.Range(func(key, value any) bool {
				req := value.(*LoginRequest)
				if req.Status == "pending" && now.Sub(req.CreatedAt) > 2*time.Minute {
					req.Status = "expired"
					s.logger.InfoModule("TELEGRAM", "Login request expired: requestId=%s ip=%s", req.ID, req.IP)
					// Update Telegram message
					newText := fmt.Sprintf(
						"⏳ <b>Yêu cầu đăng nhập đã hết hạn</b>\n\n"+
							"• <b>IP:</b> <code>%s</code>\n"+
							"• <b>Thiết bị:</b> %s\n"+
							"• <b>Thời gian tạo:</b> %s",
						htmlEscape(req.IP), htmlEscape(req.UserAgent), htmlEscape(req.CreatedAt.Local().Format("2006-01-02 15:04:05")),
					)
					go s.updateMessageStatus(req, newText)
					// Keep in map for status query briefly, but can delete later or mark as expired
				} else if now.Sub(req.CreatedAt) > 5*time.Minute {
					// Clean up requests older than 5 minutes completely
					s.requests.Delete(key)
				}
				return true
			})
		}
	}
}

// Internal structures for parsing Telegram Bot API responses
type tgUpdatesResponse struct {
	Ok     bool       `json:"ok"`
	Result []tgUpdate `json:"result"`
}

type tgUpdate struct {
	UpdateID      int              `json:"update_id"`
	Message       *tgMessage       `json:"message,omitempty"`
	CallbackQuery *tgCallbackQuery `json:"callback_query,omitempty"`
}

type tgMessage struct {
	MessageID int     `json:"message_id"`
	Chat      *tgChat `json:"chat"`
	Text      string  `json:"text,omitempty"`
	From      *tgUser `json:"from,omitempty"`
}

type tgChat struct {
	ID int64 `json:"id"`
}

type tgUser struct {
	ID int64 `json:"id"`
}

type tgCallbackQuery struct {
	ID      string     `json:"id"`
	From    *tgUser    `json:"from"`
	Message *tgMessage `json:"message,omitempty"`
	Data    string     `json:"data"`
}

func htmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}
