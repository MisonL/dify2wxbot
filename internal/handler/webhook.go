package handler

import (
	"encoding/json" // 导入 encoding/json 包，用于 JSON 数据的编解码
	"fmt"           // 导入 fmt 包，用于格式化字符串和错误信息
	"io"            // 导入 io 包，用于 IO 操作，例如读取文件内容
	"log"           // 导入 log 包，用于日志输出
	"net/http"      // 导入 net/http 包，用于处理 HTTP 请求和响应
	"os"            // 导入 os 包，用于文件操作，例如创建临时文件
	"path/filepath" // 导入 path/filepath 包，用于处理文件路径，例如获取文件名
	"strings"       // 导入 strings 包，用于字符串操作，例如检查 Content-Type 前缀

	"dify2wxbot/internal/config"  // 导入 config 包，用于加载应用程序配置
	"dify2wxbot/internal/service" // 导入 internal/service 包，包含 MessageConverter 和 DifyService
	"dify2wxbot/internal/store"   // 导入 internal/store 包，包含 ConversationStore 接口

	"github.com/google/uuid" // 导入 uuid 包，用于生成唯一标识符 (UUID)
)

// WebhookHandler 结构体定义了处理 Webhook 请求的处理器
type WebhookHandler struct {
	converter         *service.MessageConverter // converter 是一个 MessageConverter 实例，用于消息转换和发送到 Dify 及企业微信
	conversationStore store.ConversationStore   // conversationStore 用于管理用户与 Dify 之间的对话 ID，以维持上下文
	cfg               *config.AppConfig         // cfg 是应用程序配置，用于访问认证 Token 等全局设置
}

// NewWebhookHandler 创建并返回一个新的 WebhookHandler 实例
// converter: 消息转换器实例，负责消息的格式化和转发
// conversationStore: 对话存储实例，负责对话 ID 的管理
// cfg: 应用程序配置，提供必要的配置信息
func NewWebhookHandler(converter *service.MessageConverter, conversationStore store.ConversationStore, cfg *config.AppConfig) *WebhookHandler {
	return &WebhookHandler{
		converter:         converter,         // 初始化 WebhookHandler 的 converter 字段
		conversationStore: conversationStore, // 初始化 WebhookHandler 的 conversationStore 字段
		cfg:               cfg,               // 初始化 WebhookHandler 的 cfg 字段
	}
}

// HandleWebhook 处理传入的 Webhook 请求
// 该函数是整个服务的核心入口，负责接收并处理来自外部（如企业微信）或内部（如定时任务）的 HTTP POST 请求。
// 它会根据请求的 Content-Type 解析消息内容，进行认证（如果启用），管理用户对话 ID，
// 并最终通过消息转换器将消息发送到 Dify AI 服务，然后返回处理结果。
// w: http.ResponseWriter 用于写入 HTTP 响应，将处理结果返回给客户端
// r: *http.Request 包含传入的 HTTP 请求的所有信息，如方法、路径、头和请求体
func (h *WebhookHandler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	// 获取请求的远程地址 (IP:Port)，用于日志记录和追踪请求来源。
	remoteAddr := r.RemoteAddr
	// 记录接收到新 Webhook 请求的日志，包括请求方法、路径和调用方 IP 地址，便于追踪和调试。
	log.Printf("[Webhook] 接收到新请求，调用方: %s, 方法: %s, 路径: %s", remoteAddr, r.Method, r.URL.Path)

	// 强制要求请求方法为 POST。Webhook 通常通过 POST 请求发送数据。
	if r.Method != http.MethodPost {
		// 如果不是 POST 请求，返回 405 Method Not Allowed 错误，并记录日志。
		http.Error(w, "只支持 POST 请求", http.StatusMethodNotAllowed)
		log.Printf("[Webhook] 请求方法不被允许: %s", r.Method) // 记录不被允许的请求方法
		return
	}

	// --- 认证逻辑 ---
	// 检查配置文件中是否开启了认证功能 (h.cfg.EnableAuth)。
	if h.cfg.EnableAuth {
		// 从请求头中获取 Authorization 字段。
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			// 如果 Authorization 头缺失，返回 401 Unauthorized 错误，并记录日志。
			http.Error(w, "缺少 Authorization 头", http.StatusUnauthorized)
			log.Println("[Webhook] 缺少 Authorization 头")
			return
		}

		// 构造期望的 Token 格式，通常是 "Bearer YOUR_AUTH_TOKEN"。
		expectedToken := "Bearer " + h.cfg.AuthToken
		// 比较请求头中的 Token 是否与配置中预期的 Token 匹配。
		if authHeader != expectedToken {
			// 如果 Token 不匹配，返回 401 Unauthorized 错误，并记录日志。
			http.Error(w, "无效的 Token", http.StatusUnauthorized)
			log.Printf("[Webhook] Token 无效: %s", authHeader)
			return
		}
		log.Println("[Webhook] Token 认证成功")
	}

	// 定义用于存储从请求体中解析出的消息、用户、对话 ID 和文件路径的变量。
	var message string
	var user string
	var conversationID string
	var filePath string // 用于存储上传文件的临时路径

	// 获取请求的 Content-Type，用于判断请求体的格式（JSON 或 multipart/form-data）。
	contentType := r.Header.Get("Content-Type")
	log.Printf("[Webhook] 请求 Content-Type: %s", contentType)

	// --- 请求体解析逻辑 ---
	// 根据 Content-Type 处理不同类型的请求体。
	if strings.HasPrefix(contentType, "application/json") {
		// 如果 Content-Type 是 application/json，则解析 JSON 格式的请求体。
		var request struct {
			Message        string `json:"message"`         // 消息内容
			User           string `json:"user"`            // 用户标识
			ConversationID string `json:"conversation_id"` // 对话 ID
		}
		// 使用 json.NewDecoder 解码请求体到 request 结构体。
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			// 如果 JSON 解析失败，记录错误并返回 400 Bad Request。
			log.Printf("[Webhook] 解析 JSON 请求体失败: %v", err)
			http.Error(w, fmt.Sprintf("解析请求体失败: %v", err), http.StatusBadRequest)
			return
		}
		// 将解析出的值赋给相应的变量。
		message = request.Message
		user = request.User
		conversationID = request.ConversationID
		log.Printf("[Webhook] 成功解析 JSON 请求体，消息: '%s', 用户: '%s', 对话ID: '%s'", message, user, conversationID)

	} else if strings.HasPrefix(contentType, "multipart/form-data") {
		// 如果 Content-Type 是 multipart/form-data，通常用于文件上传。
		// 解析 multipart/form-data 请求，最大内存限制为 32MB。
		err := r.ParseMultipartForm(32 << 20) // 32MB
		if err != nil {
			// 如果解析失败，记录错误并返回 400 Bad Request。
			log.Printf("[Webhook] 解析 multipart/form-data 失败: %v", err)
			http.Error(w, fmt.Sprintf("解析 multipart/form-data 失败: %v", err), http.StatusBadRequest)
			return
		}

		// 从表单值中获取消息、用户和对话 ID。
		message = r.FormValue("message")
		user = r.FormValue("user")
		conversationID = r.FormValue("conversation_id")

		// 尝试获取上传的文件。
		file, handler, err := r.FormFile("file")
		if err == nil { // 如果成功获取到文件（即有文件上传）
			defer file.Close() // 确保文件在函数结束时关闭。
			// 将文件保存到系统的临时目录。
			tempDir := os.TempDir()
			filePath = filepath.Join(tempDir, handler.Filename)
			dst, createErr := os.Create(filePath) // 创建临时文件。
			if createErr != nil {
				log.Printf("[Webhook] 创建临时文件失败: %v", createErr) // 记录创建文件错误
				http.Error(w, fmt.Sprintf("创建临时文件失败: %v", createErr), http.StatusInternalServerError)
				return
			}
			defer dst.Close()         // 确保目标文件在函数结束时关闭。
			defer os.Remove(filePath) // 在处理完成后删除临时文件，避免文件残留。

			// 将上传的文件内容复制到临时文件。
			if _, copyErr := io.Copy(dst, file); copyErr != nil { // 捕获复制文件错误
				log.Printf("[Webhook] 保存临时文件失败: %v", copyErr) // 记录保存文件错误
				http.Error(w, fmt.Sprintf("保存临时文件失败: %v", copyErr), http.StatusInternalServerError)
				return
			}
			log.Printf("[Webhook] 成功接收文件: %s，保存到: %s", handler.Filename, filePath)
		} else if err != http.ErrMissingFile {
			// 如果文件获取失败，但不是因为文件缺失（即其他错误），则记录错误并返回 400 Bad Request。
			log.Printf("[Webhook] 获取文件失败: %v", err)
			http.Error(w, fmt.Sprintf("获取文件失败: %v", err), http.StatusBadRequest)
			return
		}
		log.Printf("[Webhook] 成功解析 multipart/form-data，消息: '%s', 用户: '%s', 对话ID: '%s', 文件路径: '%s'", message, user, conversationID, filePath)

	} else {
		// 如果 Content-Type 既不是 JSON 也不是 multipart/form-data，则返回 415 Unsupported Media Type 错误。
		log.Printf("[Webhook] 不支持的 Content-Type: %s", contentType)
		http.Error(w, "不支持的 Content-Type", http.StatusUnsupportedMediaType)
		return
	}

	// --- 用户和对话 ID 管理逻辑 ---
	// 如果请求中没有提供用户标识，则生成一个唯一的 UUID 作为用户标识。
	if user == "" {
		user = uuid.New().String() // 生成一个新的 UUID，确保每个请求都有一个用户标识
		log.Printf("[Webhook] 用户标识为空，生成新的用户ID: %s", user)
	}

	var currentConversationID string // 默认为空字符串

	// 如果请求中明确提供了 conversation_id，则优先使用请求中的 ID。
	if conversationID != "" {
		currentConversationID = conversationID
		// 并将此 ID 保存或更新到存储中，确保后续请求使用相同的对话上下文。
		h.conversationStore.SaveConversationID(user, currentConversationID)
		log.Printf("[Webhook] 请求中提供了对话ID '%s'，使用并更新存储。", currentConversationID)
	} else {
		// 如果请求中没有提供 conversation_id，则尝试从本地存储获取。
		storedConversationID, ok := h.conversationStore.GetConversationID(user)
		if ok {
			currentConversationID = storedConversationID
			log.Printf("[Webhook] 从存储中获取到用户 '%s' 的对话ID: %s", user, currentConversationID)
		} else {
			// 如果请求和存储中都没有提供 conversation_id，则保持 currentConversationID 为空，让 Dify 自动创建。
			log.Printf("[Webhook] 未找到用户 '%s' 的对话ID，将发送空对话ID给Dify，让Dify自动创建新会话。", user)
			// 不需要调用 h.conversationStore.NewConversationID(user)
		}
	}

	// --- 消息处理和响应 ---
	// 调用消息转换器 (h.converter) 处理并发送消息到 Dify AI 服务。
	// 传入用户标识、对话 ID 和文件路径（如果存在）。
	if err := h.converter.ConvertAndSend(message, user, currentConversationID, filePath); err != nil {
		// 如果消息处理失败（例如，与 Dify 服务通信失败），记录错误日志并返回 500 Internal Server Error。
		log.Printf("[Webhook] 处理消息失败: %v", err)
		http.Error(w, fmt.Sprintf("处理消息失败: %v", err), http.StatusInternalServerError)
		return
	}

	// 设置 HTTP 响应头，声明响应内容为 JSON 格式。
	w.Header().Set("Content-Type", "application/json")
	// 设置 HTTP 状态码为 200 OK，表示请求已成功处理。
	w.WriteHeader(http.StatusOK)
	// 构建一个表示成功响应的 JSON 结构。
	response := map[string]string{"status": "success", "message": "消息已成功处理"}
	// 将成功响应编码为 JSON 并写入 HTTP 响应体。
	if err := json.NewEncoder(w).Encode(response); err != nil {
		// 如果写入响应失败，记录错误日志。
		log.Printf("[Webhook] 写入成功响应失败: %v", err)
	}
	// 记录 Webhook 请求处理成功并返回响应的日志，表示整个处理流程完成。
	log.Println("[Webhook] 请求处理成功并返回响应")
}
