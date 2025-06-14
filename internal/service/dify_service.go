package service

import (
	"bytes"                      // 导入 bytes 包，用于处理字节缓冲区，例如构建 HTTP 请求体
	"dify2wxbot/internal/config" // 导入 config 包，用于加载应用程序配置，例如 Dify API Key 和 BaseURL
	"encoding/json"              // 导入 encoding/json 包，用于 JSON 数据的编解码
	"fmt"                        // 导入 fmt 包，用于格式化字符串和错误信息
	"io"                         // 导入 io 包，用于 IO 操作，例如读取响应体和文件内容
	"log"                        // 导入 log 包，用于日志输出
	"mime/multipart"             // 导入 mime/multipart 包，用于处理 multipart/form-data 格式的请求，主要用于文件上传
	"net/http"                   // 导入 net/http 包，用于构建和发送 HTTP 请求
	"os"                         // 导入 os 包，用于文件操作，例如打开文件
	"path/filepath"              // 导入 path/filepath 包，用于处理文件路径，例如获取文件名
	"time"                       // 导入 time 包，用于处理时间相关操作，例如设置 HTTP 客户端超时和重试间隔
)

// DifyService 结构体定义了与 Dify API 交互的服务
// 它封装了 HTTP 客户端和 Dify 相关的配置，提供了调用 Dify 各类 API 的方法。
type DifyService struct {
	httpClient *http.Client      // httpClient 是一个 HTTP 客户端实例，用于发送请求并复用连接，提高效率
	cfg        *config.AppConfig // cfg 是应用程序配置，用于获取 Dify API 相关的设置，如 API Key 和 Base URL
}

// NewDifyService 创建并返回一个新的 DifyService 实例
// cfg: 应用程序配置，用于初始化 DifyService
// 它初始化一个带有默认超时时间的 HTTP 客户端，确保 API 请求不会无限期等待。
func NewDifyService(cfg *config.AppConfig) *DifyService {
	return &DifyService{
		httpClient: &http.Client{
			Timeout: 30 * time.Second, // 设置 HTTP 请求的默认超时时间为 30 秒
		},
		cfg: cfg, // 初始化 DifyService 的 cfg 字段
	}
}

// DifyAPIErrorResponse 定义 Dify API 错误响应的结构
// 当 Dify API 返回非 200 状态码时，通常会返回此格式的错误信息。
type DifyAPIErrorResponse struct {
	Code    string `json:"code"`    // 错误码，Dify 内部定义的错误代码
	Message string `json:"message"` // 错误消息，描述具体的错误内容
	Status  int    `json:"status"`  // HTTP 状态码，例如 400, 401, 500 等
}

// DifyBaseRequest 定义 Dify API 请求的通用字段
// 这是一个嵌入结构体，包含了所有 Dify API 请求共有的字段。
type DifyBaseRequest struct {
	Inputs       map[string]interface{}   `json:"inputs"`          // 输入变量，用于传递给 Dify 应用的自定义参数
	User         string                   `json:"user"`            // 用户唯一标识，用于 Dify 区分不同用户和管理对话上下文
	ResponseMode string                   `json:"response_mode"`   // 响应模式，可以是 "blocking" (阻塞) 或 "streaming" (流式)
	Files        []map[string]interface{} `json:"files,omitempty"` // 文件列表，支持多种文件类型，`omitempty` 表示如果为空则不序列化到 JSON
}

// DifyChatRequest 定义 Dify 聊天型应用请求体
// 嵌入 DifyBaseRequest，并添加聊天应用特有的字段。
type DifyChatRequest struct {
	DifyBaseRequest
	Query          string `json:"query"`           // 用户查询文本，即用户发送给聊天机器人的消息
	ConversationID string `json:"conversation_id"` // 对话 ID，用于维持用户与 Dify 聊天机器人之间的对话上下文
}

// DifyCompletionRequest 定义 Dify 补全型应用请求体
// 嵌入 DifyBaseRequest，并添加补全应用特有的字段。
type DifyCompletionRequest struct {
	DifyBaseRequest
	Prompt string `json:"prompt"` // 补全提示词，即发送给补全模型的文本
}

// DifyWorkflowRequest 定义 Dify 工作流型应用请求体
// 嵌入 DifyBaseRequest，并添加工作流应用特有的字段。
type DifyWorkflowRequest struct {
	DifyBaseRequest
	// Workflow 应用可能没有 Query 或 Prompt 字段，主要依赖 Inputs 字段传递数据。
	WorkflowID string `json:"workflow_id"` // 工作流 ID，指定要运行的 Dify 工作流
}

// DifyChatResponse 定义 Dify 聊天型应用成功响应的结构
type DifyChatResponse struct {
	Answer string `json:"answer"` // AI 回复的答案文本
	// ... 其他聊天特有字段，根据 Dify 实际响应补充，例如 `conversation_id`, `message_id` 等
}

// DifyCompletionResponse 定义 Dify 补全型应用成功响应的结构
type DifyCompletionResponse struct {
	Text string `json:"text"` // AI 回复的补全文本
	// ... 其他补全特有字段，根据 Dify 实际响应补充
}

// DifyWorkflowResponse 定义 Dify 工作流型应用成功响应的结构
type DifyWorkflowResponse struct {
	Data map[string]interface{} `json:"data"` // 工作流执行结果数据，通常是一个 JSON 对象
	// ... 其他工作流特有字段，根据 Dify 实际响应补充
}

const (
	difyChatMessagesPath       = "/v1/chat-messages"       // Dify 聊天消息 API 的相对路径
	difyCompletionMessagesPath = "/v1/completion-messages" // Dify 补全消息 API 的相对路径
	difyWorkflowRunPath        = "/v1/workflows/run"       // Dify 工作流运行 API 的相对路径
	responseModeBlocking       = "blocking"                // Dify API 响应模式：阻塞模式，表示等待完整响应
	defaultRole                = "员工"                      // Dify API 请求中 inputs 字段的默认角色，如果未指定
	maxRetries                 = 3                         // API 请求失败时的最大重试次数
	difyFileUploadPath         = "/files/upload"           // Dify 文件上传 API 的相对路径
)

// doDifyRequest 是一个通用的辅助函数，用于发送 Dify API 请求并处理响应
// 该函数封装了 HTTP 请求的创建、发送、认证、重试机制以及错误和成功响应的解析。
// method: HTTP 方法 (e.g., "POST", "GET")
// path: Dify API 的相对路径 (e.g., "/v1/chat-messages")
// body: 请求体 (io.Reader 接口)，可以是 nil，用于 POST/PUT 请求的数据
// contentType: Content-Type 头，例如 "application/json", "multipart/form-data"
// responseStruct: 用于解析成功响应的结构体指针，如果不需要解析响应体，可以传入 nil
// logPrefix: 日志前缀，用于区分不同的 API 调用，便于日志追踪 (e.g., "Chat API", "File Upload API")
func (s *DifyService) doDifyRequest(method, path string, body io.Reader, contentType, logPrefix string, responseStruct interface{}) error {
	fullURL := fmt.Sprintf("%s%s", s.cfg.Dify.BaseURL, path) // 拼接完整的 Dify API 请求 URL
	log.Printf("[DifyService] %s 请求 URL: %s", logPrefix, fullURL)

	req, err := http.NewRequest(method, fullURL, body) // 创建新的 HTTP 请求
	if err != nil {
		return fmt.Errorf("failed to create %s http request: %w", logPrefix, err) // 如果请求创建失败，返回错误
	}

	req.Header.Set("Content-Type", contentType)                  // 设置请求的 Content-Type 头
	req.Header.Set("Authorization", "Bearer "+s.cfg.Dify.APIKey) // 设置 Authorization 头，携带 Dify API Key 进行认证

	var resp *http.Response // 用于存储 HTTP 响应
	// 循环重试机制，最多重试 maxRetries 次
	for i := 0; i < maxRetries; i++ {
		resp, err = s.httpClient.Do(req) // 使用 DifyService 的 HTTP 客户端发送请求
		if err == nil {                  // 如果请求成功（没有网络错误），则跳出重试循环
			break
		}
		log.Printf("[DifyService] %s 请求失败，正在重试 %d/%d 次: %v", logPrefix, i+1, maxRetries, err)
		if i < maxRetries-1 { // 如果不是最后一次重试，则等待一段时间再重试
			time.Sleep(time.Duration(i+1) * time.Second) // 每次重试等待时间递增
		}
	}
	if err != nil {
		return fmt.Errorf("%s 请求在 %d 次重试后仍然失败: %w", logPrefix, maxRetries, err) // 如果所有重试都失败，返回错误
	}
	defer resp.Body.Close() // 确保在函数返回前关闭响应体，释放资源

	respBody, err := io.ReadAll(resp.Body) // 读取完整的响应体内容
	if err != nil {
		return fmt.Errorf("failed to read %s 响应体: %w", logPrefix, err) // 如果读取响应体失败，返回错误
	}

	log.Printf("[DifyService] %s 响应状态码: %d", logPrefix, resp.StatusCode) // 记录响应状态码
	log.Printf("[DifyService] %s 响应体: %s", logPrefix, string(respBody))  // 记录完整的响应体内容

	// 检查 HTTP 状态码是否为 200 OK
	if resp.StatusCode != http.StatusOK {
		var errorResponse DifyAPIErrorResponse
		// 尝试将错误响应体解析为 DifyAPIErrorResponse 结构体
		if err := json.Unmarshal(respBody, &errorResponse); err == nil {
			// 如果解析成功，返回 Dify 提供的具体错误信息
			return fmt.Errorf("%s 错误: 错误码: %s, 消息: %s", logPrefix, errorResponse.Code, errorResponse.Message)
		}
		// 如果解析失败，返回通用的错误状态码和原始响应体
		return fmt.Errorf("%s 返回错误状态码 %d: %s", logPrefix, resp.StatusCode, string(respBody))
	}

	// 如果提供了 responseStruct，则将成功响应体解析到该结构体
	if responseStruct != nil {
		if err := json.Unmarshal(respBody, responseStruct); err != nil {
			return fmt.Errorf("failed to parse %s 成功响应: %w", logPrefix, err) // 如果解析失败，返回错误
		}
	}

	log.Printf("[DifyService] %s 调用成功。", logPrefix) // 记录 API 调用成功日志
	return nil
}

// CallDifyChatAPI 调用 Dify 聊天型应用 API 发送消息并获取回复
// request: DifyChatRequest 结构体，包含查询文本、输入变量、用户标识和对话 ID
func (s *DifyService) CallDifyChatAPI(request DifyChatRequest) (DifyChatResponse, error) {
	log.Printf("[DifyService] 调用 Chat API，用户: '%s', 对话ID: '%s', 查询: '%s'", request.User, request.ConversationID, request.Query)
	// 检查 Dify Base URL 和 API Key 是否已配置
	if s.cfg.Dify.BaseURL == "" || s.cfg.Dify.APIKey == "" {
		return DifyChatResponse{}, fmt.Errorf("dify base url 或 api key 未配置")
	}

	// 确保 inputs 中包含 "role" 字段，如果不存在则使用默认角色
	if request.Inputs == nil {
		request.Inputs = make(map[string]interface{})
	}
	if _, ok := request.Inputs["role"]; !ok {
		request.Inputs["role"] = defaultRole
	}

	// 设置响应模式为阻塞，确保获取完整回复
	request.ResponseMode = responseModeBlocking

	jsonData, err := json.Marshal(request) // 将请求结构体编码为 JSON 字节
	if err != nil {
		return DifyChatResponse{}, fmt.Errorf("failed to marshal chat request body: %w", err) // 如果编码失败，返回错误
	}

	var response DifyChatResponse // 用于存储 Dify 聊天 API 的成功响应
	err = s.doDifyRequest(
		"POST",                    // HTTP 方法为 POST
		difyChatMessagesPath,      // 聊天消息 API 的相对路径
		bytes.NewBuffer(jsonData), // 请求体为 JSON 数据
		"application/json",        // Content-Type 为 application/json
		"Chat API",                // 日志前缀
		&response,                 // 响应解析目标
	)
	if err != nil {
		return DifyChatResponse{}, err // 如果 doDifyRequest 失败，返回错误
	}

	// 检查 Dify 响应中是否包含有效的答案
	if response.Answer == "" {
		return DifyChatResponse{}, fmt.Errorf("dify chat api 响应未包含有效答案")
	}

	return response, nil // 返回成功响应
}

// DownloadFile 从指定的 URL 下载文件并保存到本地路径
// fileURL: 文件的远程 URL
// outputPath: 文件保存的本地路径
func (s *DifyService) DownloadFile(fileURL, outputPath string) error {
	log.Printf("[DifyService] 尝试从 URL '%s' 下载文件到 '%s'", fileURL, outputPath)

	resp, err := s.httpClient.Get(fileURL) // 发送 GET 请求下载文件
	if err != nil {
		return fmt.Errorf("failed to download file from %s: %w", fileURL, err) // 如果下载失败，返回错误
	}
	defer resp.Body.Close() // 确保在函数返回前关闭响应体

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download file, received status code %d from %s", resp.StatusCode, fileURL) // 如果状态码不是 200 OK，返回错误
	}

	out, err := os.Create(outputPath) // 创建本地文件用于写入下载内容
	if err != nil {
		return fmt.Errorf("failed to create local file %s: %w", outputPath, err) // 如果文件创建失败，返回错误
	}
	defer out.Close() // 确保在函数返回前关闭文件

	_, err = io.Copy(out, resp.Body) // 将下载内容从响应体复制到本地文件
	if err != nil {
		return fmt.Errorf("failed to write downloaded file to %s: %w", outputPath, err) // 如果写入文件失败，返回错误
	}

	log.Printf("[DifyService] 文件成功下载到 '%s'", outputPath) // 记录文件下载成功日志
	return nil
}

// UploadFile 上传文件到 Dify
// filePath: 本地文件路径，待上传的文件在本地文件系统中的路径
// user: 用户唯一标识，用于 Dify 关联文件上传和用户
func (s *DifyService) UploadFile(filePath, user string) (map[string]interface{}, error) {
	log.Printf("[DifyService] 尝试上传文件 '%s' 到 Dify，用户: '%s'", filePath, user)
	// 检查 Dify Base URL 和 API Key 是否已配置
	if s.cfg.Dify.BaseURL == "" || s.cfg.Dify.APIKey == "" {
		return nil, fmt.Errorf("dify base url 或 api key 未配置")
	}

	file, err := os.Open(filePath) // 打开本地文件
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err) // 如果文件打开失败，返回错误
	}
	defer file.Close() // 确保在函数返回时关闭文件

	body := &bytes.Buffer{}             // 创建一个字节缓冲区用于构建 multipart 请求体
	writer := multipart.NewWriter(body) // 创建 multipart 写入器

	// 创建文件表单字段，将文件内容写入请求体
	part, err := writer.CreateFormFile("file", filepath.Base(filePath)) // "file" 是 Dify API 期望的文件字段名
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err) // 如果创建表单文件失败，返回错误
	}
	_, err = io.Copy(part, file) // 将文件内容复制到表单字段中
	if err != nil {
		return nil, fmt.Errorf("failed to copy file content: %w", err) // 如果复制文件内容失败，返回错误
	}

	// 确保 user 字段被正确写入 multipart 表单
	if err := writer.WriteField("user", user); err != nil {
		return nil, fmt.Errorf("failed to write user field to form: %w", err) // 如果写入用户字段失败，返回错误
	}
	writer.Close() // 关闭 multipart 写入器，完成请求体的构建

	var response map[string]interface{} // 用于存储 Dify 文件上传 API 的成功响应
	err = s.doDifyRequest(
		"POST",                       // HTTP 方法为 POST
		difyFileUploadPath,           // 文件上传 API 的相对路径
		body,                         // 请求体为 multipart 数据
		writer.FormDataContentType(), // Content-Type 为 multipart/form-data
		"File Upload API",            // 日志前缀
		&response,                    // 响应解析目标
	)
	if err != nil {
		return nil, err // 如果 doDifyRequest 失败，返回错误
	}

	log.Println("[DifyService] 文件上传成功。") // 记录文件上传成功日志
	return response, nil                 // 返回成功响应
}

// CallDifyCompletionAPI 调用 Dify 补全型应用 API 发送消息并获取回复
// request: DifyCompletionRequest 结构体，包含提示词、输入变量、用户标识
func (s *DifyService) CallDifyCompletionAPI(request DifyCompletionRequest) (DifyCompletionResponse, error) {
	log.Printf("[DifyService] 调用 Completion API，用户: '%s', 提示词: '%s'", request.User, request.Prompt)
	// 检查 Dify Base URL 和 API Key 是否已配置
	if s.cfg.Dify.BaseURL == "" || s.cfg.Dify.APIKey == "" {
		return DifyCompletionResponse{}, fmt.Errorf("dify base url 或 api key 未配置")
	}

	// 确保 inputs 中包含 "role" 字段，如果不存在则使用默认角色
	if request.Inputs == nil {
		request.Inputs = make(map[string]interface{})
	}
	if _, ok := request.Inputs["role"]; !ok {
		request.Inputs["role"] = defaultRole
	}

	// 设置响应模式为阻塞，确保获取完整回复
	request.ResponseMode = responseModeBlocking

	jsonData, err := json.Marshal(request) // 将请求结构体编码为 JSON 字节
	if err != nil {
		return DifyCompletionResponse{}, fmt.Errorf("failed to marshal completion request body: %w", err) // 如果编码失败，返回错误
	}

	var response DifyCompletionResponse // 用于存储 Dify 补全 API 的成功响应
	err = s.doDifyRequest(
		"POST",                     // HTTP 方法为 POST
		difyCompletionMessagesPath, // 补全消息 API 的相对路径
		bytes.NewBuffer(jsonData),  // 请求体为 JSON 数据
		"application/json",         // Content-Type 为 application/json
		"Completion API",           // 日志前缀
		&response,                  // 响应解析目标
	)
	if err != nil {
		return DifyCompletionResponse{}, err // 如果 doDifyRequest 失败，返回错误
	}

	// 检查 Dify 响应中是否包含有效的文本
	if response.Text == "" {
		return DifyCompletionResponse{}, fmt.Errorf("dify completion api 响应未包含有效文本")
	}

	return response, nil // 返回成功响应
}

// CallDifyWorkflowAPI 调用 Dify 工作流型应用 API 运行工作流并获取结果
// request: DifyWorkflowRequest 结构体，包含输入变量、用户标识和工作流 ID
func (s *DifyService) CallDifyWorkflowAPI(request DifyWorkflowRequest) (DifyWorkflowResponse, error) {
	log.Printf("[DifyService] 调用 Workflow API，用户: '%s', 工作流ID: '%s'", request.User, request.WorkflowID)
	// 检查 Dify Base URL 和 API Key 是否已配置
	if s.cfg.Dify.BaseURL == "" || s.cfg.Dify.APIKey == "" {
		return DifyWorkflowResponse{}, fmt.Errorf("dify base url 或 api key 未配置")
	}

	// 确保 inputs 不为空，工作流通常依赖 inputs 传递数据
	if request.Inputs == nil {
		request.Inputs = make(map[string]interface{})
	}

	// 设置响应模式为阻塞，确保获取完整结果
	request.ResponseMode = responseModeBlocking

	jsonData, err := json.Marshal(request) // 将请求结构体编码为 JSON 字节
	if err != nil {
		return DifyWorkflowResponse{}, fmt.Errorf("failed to marshal workflow request body: %w", err) // 如果编码失败，返回错误
	}

	var response DifyWorkflowResponse // 用于存储 Dify 工作流 API 的成功响应
	err = s.doDifyRequest(
		"POST",                    // HTTP 方法为 POST
		difyWorkflowRunPath,       // 工作流运行 API 的相对路径
		bytes.NewBuffer(jsonData), // 请求体为 JSON 数据
		"application/json",        // Content-Type 为 application/json
		"Workflow API",            // 日志前缀
		&response,                 // 响应解析目标
	)
	if err != nil {
		return DifyWorkflowResponse{}, err // 如果 doDifyRequest 失败，返回错误
	}

	// 检查 Dify 响应中是否包含有效数据
	if response.Data == nil {
		return DifyWorkflowResponse{}, fmt.Errorf("dify workflow api 响应未包含有效数据")
	}

	return response, nil // 返回成功响应
}
