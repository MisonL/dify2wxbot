package service

import (
	"dify2wxbot/internal/config" // 导入 config 包，用于获取应用程序配置，例如 Dify API 的 BotType 和 DefaultPrompt
	"dify2wxbot/pkg/wecom"       // 导入 pkg/wecom 包，用于与企业微信机器人交互，发送消息
	"encoding/json"              // 导入 encoding/json 包，用于 JSON 数据的编解码，例如处理工作流响应
	"fmt"                        // 导入 fmt 包，用于格式化字符串和错误信息
	"log"                        // 导入 log 包，用于日志输出
	"os"                         // 导入 os 包，用于文件操作，例如创建临时文件和删除文件
	"path/filepath"              // 导入 path/filepath 包，用于处理文件路径，例如获取文件扩展名
	"strings"                    // 导入 strings 包，用于字符串操作，例如将文件扩展名转换为小写
)

// MessageConverter 结构体定义了消息转换和发送的服务
// 它负责将接收到的消息（可能包含文件）发送到 Dify AI 服务进行处理，
// 然后将 Dify 的回复转换并发送到企业微信机器人。
type MessageConverter struct {
	robot       *wecom.Robot // robot 是一个企业微信机器人实例，用于发送消息到企业微信群
	difyService *DifyService // difyService 是一个 DifyService 实例，用于与 Dify API 交互
}

// NewMessageConverter 创建并返回一个新的 MessageConverter 实例
// cfg: 应用程序配置，用于初始化企业微信机器人
// difyService: Dify 服务实例，用于与 Dify AI 交互
func NewMessageConverter(cfg *config.AppConfig, difyService *DifyService) *MessageConverter {
	return &MessageConverter{
		robot:       wecom.NewRobot(cfg), // 使用配置创建并初始化企业微信机器人实例
		difyService: difyService,         // 初始化 Dify 服务实例
	}
}

// preprocessMessage 对用户消息进行预处理，例如识别特定命令
// message: 原始用户消息
// 返回值：处理后的消息，是否已处理（如果为 true，则不再调用 Dify），错误
func (c *MessageConverter) preprocessMessage(message string) (string, bool, error) {
	// 示例：如果消息以 "/image" 开头，可以尝试生成图片或执行特定逻辑
	if strings.HasPrefix(message, "/image ") {
		// 这里可以添加调用图片生成 AI 的逻辑
		// 为了演示，我们简单地返回一个提示
		log.Printf("[Converter] 识别到图片生成命令: '%s'", message)
		// 假设我们直接回复用户，不调用 Dify
		return "抱歉，图片生成功能暂未实现。", true, nil
	}
	// 其他预处理逻辑...

	return message, false, nil // 默认情况下，不处理消息，继续调用 Dify
}

// postprocessDifyResponse 对 Dify 的响应进行后处理，根据内容发送不同类型的企业微信消息
// difyResponse: Dify API 的原始响应字符串
func (c *MessageConverter) postprocessDifyResponse(difyResponse string) error {
	log.Printf("[Converter] 开始后处理 Dify 响应，长度: %d", len(difyResponse))

	// 尝试将 Dify 响应解析为 JSON，以便检查是否有结构化数据（如图片URL、文件URL）
	var jsonResponse map[string]interface{}
	if err := json.Unmarshal([]byte(difyResponse), &jsonResponse); err == nil {
		// 检查是否有图片 URL
		if imageUrl, ok := jsonResponse["image_url"].(string); ok && imageUrl != "" {
			log.Printf("[Converter] Dify 响应包含图片 URL: %s", imageUrl)
			// 获取图片文件扩展名
			imageExt := filepath.Ext(imageUrl)
			if imageExt == "" {
				imageExt = ".png" // 默认图片扩展名
			}
			// 下载图片到本地临时文件，并保留扩展名
			tempFile, err := os.CreateTemp("", "dify_image_*"+imageExt)
			if err != nil {
				log.Printf("[Converter] 创建临时图片文件失败: %v", err)
				return c.robot.SendTextMessage(fmt.Sprintf("Dify 返回了一张图片: %s，但下载失败。", imageUrl))
			}
			tempFilePath := tempFile.Name()
			tempFile.Close() // 关闭文件句柄，以便 DifyService.DownloadFile 可以写入

			if err := c.difyService.DownloadFile(imageUrl, tempFilePath); err != nil {
				os.Remove(tempFilePath) // 下载失败，删除临时文件
				log.Printf("[Converter] 下载 Dify 图片失败: %v", err)
				return c.robot.SendTextMessage(fmt.Sprintf("Dify 返回了一张图片: %s，但下载失败。", imageUrl))
			}
			defer os.Remove(tempFilePath) // 确保函数退出时删除临时文件

			// 发送图片消息到企业微信
			if err := c.robot.SendImageMessage(tempFilePath); err != nil {
				log.Printf("[Converter] 发送图片消息到企业微信失败: %v", err)
				return c.robot.SendTextMessage(fmt.Sprintf("Dify 返回了一张图片: %s，但发送失败。", imageUrl))
			}
			return nil // 图片消息已发送，不再发送文本
		}
		// 检查是否有文件 URL
		if fileUrl, ok := jsonResponse["file_url"].(string); ok && fileUrl != "" {
			log.Printf("[Converter] Dify 响应包含文件 URL: %s", fileUrl)
			// 获取文件扩展名
			fileExt := filepath.Ext(fileUrl)
			if fileExt == "" {
				fileExt = ".bin" // 默认二进制文件扩展名
			}
			// 下载文件到本地临时文件，并保留扩展名
			tempFile, err := os.CreateTemp("", "dify_file_*"+fileExt)
			if err != nil {
				log.Printf("[Converter] 创建临时文件失败: %v", err)
				return c.robot.SendTextMessage(fmt.Sprintf("Dify 返回了一个文件: %s，但下载失败。", fileUrl))
			}
			tempFilePath := tempFile.Name()
			tempFile.Close() // 关闭文件句柄，以便 DifyService.DownloadFile 可以写入

			if err := c.difyService.DownloadFile(fileUrl, tempFilePath); err != nil {
				os.Remove(tempFilePath) // 下载失败，删除临时文件
				log.Printf("[Converter] 下载 Dify 文件失败: %v", err)
				return c.robot.SendTextMessage(fmt.Sprintf("Dify 返回了一个文件: %s，但下载失败。", fileUrl))
			}
			defer os.Remove(tempFilePath) // 确保函数退出时删除临时文件

			// 发送文件消息到企业微信
			if err := c.robot.SendFileMessage(tempFilePath); err != nil {
				log.Printf("[Converter] 发送文件消息到企业微信失败: %v", err)
				return c.robot.SendTextMessage(fmt.Sprintf("Dify 返回了一个文件: %s，但发送失败。", fileUrl))
			}
			return nil // 文件消息已发送，不再发送文本
		}
		// 检查是否有 Markdown 内容
		if markdownContent, ok := jsonResponse["markdown"].(string); ok && markdownContent != "" {
			log.Printf("[Converter] Dify 响应包含 Markdown 内容，长度: %d", len(markdownContent))
			return c.robot.SendMarkdownMessage(markdownContent)
		}
		// 如果是工作流响应，并且是 JSON 格式，可以考虑发送为 Markdown 或文本
		if _, ok := jsonResponse["data"]; ok && c.difyService.cfg.Dify.BotType == "workflow" {
			log.Printf("[Converter] Dify Workflow 响应为 JSON 格式，将作为文本发送。")
			// 已经处理过截断，直接发送
			return c.robot.SendTextMessage(difyResponse)
		}
	}

	// 如果不是结构化响应，或者没有识别到特定类型，则作为普通文本消息发送
	log.Printf("[Converter] Dify 响应为纯文本或无法解析，将作为文本发送。")

	// 企业微信机器人文本消息最大长度为 2048 字节
	const maxWeComMessageLength = 2048
	if len(difyResponse) > maxWeComMessageLength {
		log.Printf("[Converter] Dify 回复长度 (%d 字节) 超过企业微信消息限制 (%d 字节)，将进行截断。", len(difyResponse), maxWeComMessageLength)
		// 截断消息并添加提示信息
		truncatedResponse := []rune(difyResponse)[:maxWeComMessageLength-50] // 预留 50 字符用于提示信息
		difyResponse = string(truncatedResponse) + "\n... (消息已截断，请查看 Dify 后台获取完整内容)"
	}

	return c.robot.SendTextMessage(difyResponse)
}

// ConvertAndSend 方法用于转换消息并将其发送到企业微信机器人
// 这是消息处理的核心逻辑，根据 Dify Bot 类型和是否包含文件进行不同的 API 调用。
// message: 待发送的原始消息字符串，可以是用户输入或定时任务的默认消息
// user: 用户标识，用于 Dify API 请求和对话上下文管理
// conversationID: 对话 ID，用于维持用户与 Dify 之间的对话上下文
// filePath: 上传文件的本地路径 (如果存在)，用于文件上传到 Dify
func (c *MessageConverter) ConvertAndSend(message, user, conversationID, filePath string) error {
	log.Printf("[Converter] 开始处理消息，用户: '%s', 对话ID: '%s', 消息: '%s', 文件路径: '%s'", user, conversationID, message, filePath)

	// 1. 消息预处理
	processedMessage, handled, err := c.preprocessMessage(message)
	if err != nil {
		return fmt.Errorf("message preprocessing failed: %w", err)
	}
	if handled {
		log.Printf("[Converter] 消息已在预处理阶段处理，直接返回。")
		// 如果预处理函数已经发送了消息或处理了逻辑，则直接返回
		// 这里的 processedMessage 可能是预处理后的回复，需要发送
		if processedMessage != "" {
			return c.robot.SendTextMessage(processedMessage)
		}
		return nil
	}
	message = processedMessage // 使用预处理后的消息

	// 如果消息为空且配置了默认提示词，则使用默认提示词
	if message == "" && c.difyService.cfg.Dify.DefaultPrompt != "" {
		message = c.difyService.cfg.Dify.DefaultPrompt
		log.Printf("[Converter] 消息为空，使用默认提示词: '%s'", message)
	}

	// 如果消息仍然为空（即没有传入消息也没有配置默认提示词）且没有文件路径，则返回错误
	if message == "" && filePath == "" {
		return fmt.Errorf("message content or file path cannot be empty")
	}

	// 根据配置的 BotType 调用不同的 Dify API
	var difyResponse string // 用于存储 Dify API 的回复内容
	var difyErr error       // 用于捕获 API 调用过程中可能发生的错误

	log.Printf("[Converter] 调用 Dify API，Bot 类型: %s", c.difyService.cfg.Dify.BotType)
	switch c.difyService.cfg.Dify.BotType {
	case "chat": // 如果 Bot 类型是 "chat" (聊天型应用)
		var files []map[string]interface{} // 用于存储上传到 Dify 的文件信息
		if filePath != "" {                // 如果存在文件路径，则先上传文件
			log.Printf("[Converter] 正在上传文件 '%s' 到 Dify...", filePath)
			uploadResp, uploadErr := c.difyService.UploadFile(filePath, user) // 调用 DifyService 上传文件
			if uploadErr != nil {
				difyErr = fmt.Errorf("failed to upload file to Dify: %w", uploadErr) // 文件上传失败则返回错误
				break                                                                // 跳出 switch
			}
			// 检查上传响应中是否包含文件 ID
			if fileID, ok := uploadResp["id"].(string); ok {
				fileType := getFileTypeFromPath(filePath)     // 根据文件路径获取文件类型
				files = append(files, map[string]interface{}{ // 构建文件信息结构
					"type":            fileType,     // 文件类型 (e.g., "image", "audio")
					"transfer_method": "local_file", // 传输方法
					"upload_file_id":  fileID,       // 上传后 Dify 返回的文件 ID
				})
				log.Printf("[Converter] 文件上传成功，文件ID: %s, 类型: %s", fileID, fileType)
			} else {
				log.Printf("[Converter] 文件上传成功但未获取到文件ID") // 如果没有获取到文件 ID，记录警告
			}
		}

		// 构建 Dify 聊天请求体
		req := DifyChatRequest{
			DifyBaseRequest: DifyBaseRequest{
				Inputs:       map[string]interface{}{}, // 根据 Dify 应用的配置填充 inputs
				User:         user,                     // 用户标识
				ResponseMode: responseModeBlocking,     // 响应模式为阻塞
				Files:        files,                    // 包含上传的文件列表
			},
			Query:          message,        // 用户查询文本
			ConversationID: conversationID, // 对话 ID
		}
		resp, e := c.difyService.CallDifyChatAPI(req) // 调用 Dify 聊天 API
		if e != nil {
			difyErr = fmt.Errorf("dify chat api call failed: %w", e) // 如果调用失败，设置错误
		} else {
			difyResponse = resp.Answer // 获取 Dify 的回答
			log.Printf("[Converter] Dify Chat API 响应成功，回答长度: %d", len(difyResponse))
		}
	case "completion": // 如果 Bot 类型是 "completion" (补全型应用)
		// 构建 Dify 补全请求体
		req := DifyCompletionRequest{
			DifyBaseRequest: DifyBaseRequest{
				Inputs:       map[string]interface{}{}, // 根据 Dify 应用的配置填充 inputs
				User:         user,                     // 用户标识
				ResponseMode: responseModeBlocking,     // 响应模式为阻塞
			},
			Prompt: message, // 补全提示词
		}
		resp, e := c.difyService.CallDifyCompletionAPI(req) // 调用 Dify 补全 API
		if e != nil {
			difyErr = fmt.Errorf("dify completion api call failed: %w", e) // 如果调用失败，设置错误
		} else {
			difyResponse = resp.Text // 获取 Dify 的补全文本
			log.Printf("[Converter] Dify Completion API 响应成功，文本长度: %d", len(difyResponse))
		}
	case "workflow": // 如果 Bot 类型是 "workflow" (工作流型应用)
		// 构建 Dify 工作流请求体
		req := DifyWorkflowRequest{
			DifyBaseRequest: DifyBaseRequest{
				Inputs:       map[string]interface{}{"query": message}, // 工作流通常通过 inputs 字段传递数据
				User:         user,                                     // 用户标识
				ResponseMode: responseModeBlocking,                     // 响应模式为阻塞
			},
			WorkflowID: c.difyService.cfg.Dify.WorkflowID, // 工作流 ID，从配置中获取
		}
		resp, e := c.difyService.CallDifyWorkflowAPI(req) // 调用 Dify 工作流 API
		if e != nil {
			difyErr = fmt.Errorf("dify workflow api call failed: %w", e) // 如果调用失败，设置错误
		} else {
			// 工作流的响应可能是一个复杂的数据结构，这里将其序列化为 JSON 字符串
			jsonBytes, marshalErr := json.MarshalIndent(resp.Data, "", "  ") // 格式化输出 JSON
			if marshalErr != nil {
				log.Printf("[Converter] 序列化 Dify Workflow 响应失败: %v", marshalErr)
				difyResponse = fmt.Sprintf("Error: failed to marshal workflow response: %v", marshalErr)
			} else {
				difyResponse = string(jsonBytes) // 将 JSON 字节转换为字符串
			}
			log.Printf("[Converter] Dify Workflow API 响应成功，数据长度: %d", len(difyResponse))
		}
	default: // 如果 Bot 类型不支持
		difyErr = fmt.Errorf("unsupported dify bot type: %s", c.difyService.cfg.Dify.BotType) // 返回不支持的 Bot 类型错误
	}

	// 如果 Dify API 调用过程中发生错误，则返回该错误
	if difyErr != nil {
		return fmt.Errorf("failed to call Dify API: %w", difyErr)
	}

	// 2. Dify 响应后处理并发送到企业微信
	err = c.postprocessDifyResponse(difyResponse)
	if err != nil {
		return fmt.Errorf("failed to post-process Dify response and send to wecom: %w", err)
	}

	log.Println("[Converter] 消息成功发送到企业微信。")
	return nil // 消息成功发送，返回 nil
}

// getFileTypeFromPath 根据文件路径判断文件类型，返回 Dify API 期望的类型字符串
// filePath: 文件的完整路径
func getFileTypeFromPath(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath)) // 获取文件扩展名并转换为小写，例如 ".jpg" -> "jpg"
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".bmp", ".webp": // 常见图片格式
		return "image"
	case ".mp3", ".wav", ".aac", ".flac": // 常见音频格式
		return "audio"
	case ".mp4", ".avi", ".mov", ".wmv", ".flv": // 常见视频格式
		return "video"
	case ".txt", ".md", ".csv", ".json", ".xml": // 常见文本格式
		return "text"
	default:
		return "other" // 默认返回 "other" 类型，表示不识别的或通用文件类型
	}
}
