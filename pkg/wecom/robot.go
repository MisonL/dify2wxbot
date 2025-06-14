package wecom

import (
	"bytes"          // 导入 bytes 包，用于处理字节缓冲区，例如构建 HTTP 请求体
	"encoding/json"  // 导入 encoding/json 包，用于 JSON 数据的编解码
	"fmt"            // 导入 fmt 包，用于格式化字符串和错误信息
	"io"             // 导入 io 包，用于 IO 操作，例如读取文件内容
	"log"            // 导入 log 包，用于日志输出
	"mime/multipart" // 导入 mime/multipart 包，用于处理 multipart/form-data 格式的请求
	"net/http"       // 导入 net/http 包，用于构建和发送 HTTP 请求
	"net/url"        // 导入 net/url 包，用于 URL 的解析和操作
	"os"             // 导入 os 包，用于文件操作，例如打开文件
	"path/filepath"  // 导入 path/filepath 包，用于处理文件路径
	"time"           // 导入 time 包，用于处理时间相关操作

	"dify2wxbot/internal/config" // 导入 config 包，用于加载应用程序配置
)

// Robot 结构体定义了企业微信机器人的客户端
type Robot struct {
	cfg        *config.AppConfig // cfg 存储应用程序的配置，包含企业微信 Webhook URL
	httpClient *http.Client      // httpClient 是一个 HTTP 客户端实例，用于发送请求并复用连接
}

// NewRobot 创建并返回一个新的 Robot 实例
// cfg: 应用程序配置，包含企业微信 Webhook URL
func NewRobot(cfg *config.AppConfig) *Robot {
	return &Robot{
		cfg: cfg, // 初始化 Robot 的 cfg 字段
		httpClient: &http.Client{
			Timeout: 10 * time.Second, // 设置 HTTP 请求的默认超时时间为 10 秒
		},
	}
}

// getWebhookKey 从企业微信 Webhook URL 中提取 'key' 参数
func (r *Robot) getWebhookKey() (string, error) {
	parsedURL, err := url.Parse(r.cfg.WeCom.WebhookURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse wecom webhook url: %w", err)
	}
	key := parsedURL.Query().Get("key")
	if key == "" {
		return "", fmt.Errorf("wecom webhook url is missing 'key' parameter")
	}
	return key, nil
}

// uploadMedia 上传媒体文件到企业微信，并返回 media_id
// mediaFilePath: 媒体文件的本地路径
// mediaType: 媒体类型，例如 "image", "voice", "video", "file"
func (r *Robot) uploadMedia(mediaFilePath, mediaType string) (string, error) {
	log.Printf("[WeCom Robot] 尝试上传媒体文件 '%s' (类型: %s) 到企业微信...", mediaFilePath, mediaType)

	key, err := r.getWebhookKey()
	if err != nil {
		return "", err
	}

	uploadURL := fmt.Sprintf("https://qyapi.weixin.qq.com/cgi-bin/webhook/upload_media?key=%s&type=%s", key, mediaType)

	file, err := os.Open(mediaFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to open media file: %w", err)
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("media", filepath.Base(mediaFilePath))
	if err != nil {
		return "", fmt.Errorf("failed to create form file for media: %w", err)
	}
	_, err = io.Copy(part, file)
	if err != nil {
		return "", fmt.Errorf("failed to copy media file content: %w", err)
	}
	writer.Close()

	req, err := http.NewRequest(http.MethodPost, uploadURL, body)
	if err != nil {
		return "", fmt.Errorf("failed to create media upload request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send media upload request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read media upload response body: %w", err)
	}

	var result struct {
		ErrCode   int    `json:"errcode"`
		ErrMsg    string `json:"errmsg"`
		MediaID   string `json:"media_id"`
		Type      string `json:"type"`
		CreatedAt string `json:"created_at"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("failed to parse media upload response: %w, body: %s", err, string(respBody))
	}

	if result.ErrCode != 0 {
		return "", fmt.Errorf("wecom media upload failed: %s (errcode: %d)", result.ErrMsg, result.ErrCode)
	}

	log.Printf("[WeCom Robot] 媒体文件上传成功，MediaID: %s", result.MediaID)
	return result.MediaID, nil
}

// sendMessageToWeCom 是一个通用的辅助函数，用于向企业微信机器人发送消息
func (r *Robot) sendMessageToWeCom(msgType string, payload interface{}) error {
	log.Printf("[WeCom Robot] 尝试发送 %s 类型消息到企业微信...", msgType)

	msg := map[string]interface{}{
		"msgtype": msgType,
		msgType:   payload,
	}

	jsonData, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal %s message: %w", msgType, err)
	}

	resp, err := r.httpClient.Post(r.cfg.WeCom.WebhookURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send %s message: %w", msgType, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			log.Printf("[WeCom Robot] 读取企业微信响应体失败: %v", readErr)
			return fmt.Errorf("failed to send %s message (status code %d), could not read response body: %w", msgType, resp.StatusCode, readErr)
		}
		return fmt.Errorf("failed to send %s message (status code %d): %s", msgType, resp.StatusCode, string(body))
	}

	var result struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	respBody, err := io.ReadAll(resp.Body) // 读取响应体
	if err != nil {
		return fmt.Errorf("failed to read %s response body: %w", msgType, err)
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return fmt.Errorf("failed to parse %s response: %w, body: %s", msgType, err, string(respBody))
	}

	if result.ErrCode != 0 {
		if result.ErrCode == 45009 { // 45009 错误码通常表示 API 调用频率超过限制
			log.Printf("[WeCom Robot] 警告: 企业微信消息发送频率限制，错误码: %d, 消息: %s", result.ErrCode, result.ErrMsg)
			return fmt.Errorf("wecom %s message failed due to rate limit: %s (errcode: %d)", msgType, result.ErrMsg, result.ErrCode)
		}
		return fmt.Errorf("wecom %s message failed: %s (errcode: %d)", msgType, result.ErrMsg, result.ErrCode)
	}

	log.Printf("[WeCom Robot] %s 消息成功发送到企业微信。", msgType)
	return nil
}

// SendTextMessage 向企业微信机器人发送文本消息
// message: 文本消息内容
func (r *Robot) SendTextMessage(message string) error {
	payload := struct {
		Content string `json:"content"`
	}{
		Content: message,
	}
	return r.sendMessageToWeCom("text", payload)
}

// SendMarkdownMessage 向企业微信机器人发送 Markdown 消息
// content: Markdown 格式的内容
func (r *Robot) SendMarkdownMessage(content string) error {
	payload := struct {
		Content string `json:"content"`
	}{
		Content: content,
	}
	return r.sendMessageToWeCom("markdown", payload)
}

// SendMarkdownV2Message 向企业微信机器人发送 Markdown V2 消息
// content: Markdown V2 格式的内容
func (r *Robot) SendMarkdownV2Message(content string) error {
	payload := struct {
		Content string `json:"content"`
	}{
		Content: content,
	}
	return r.sendMessageToWeCom("markdown_v2", payload)
}

// SendImageMessage 向企业微信机器人发送图片消息
// imageFilePath: 图片文件的本地路径
func (r *Robot) SendImageMessage(imageFilePath string) error {
	mediaID, err := r.uploadMedia(imageFilePath, "image")
	if err != nil {
		return fmt.Errorf("failed to upload image for WeCom: %w", err)
	}
	payload := struct {
		MediaID string `json:"media_id"`
	}{
		MediaID: mediaID,
	}
	return r.sendMessageToWeCom("image", payload)
}

// SendVoiceMessage 向企业微信机器人发送语音消息
// voiceFilePath: 语音文件的本地路径
func (r *Robot) SendVoiceMessage(voiceFilePath string) error {
	mediaID, err := r.uploadMedia(voiceFilePath, "voice")
	if err != nil {
		return fmt.Errorf("failed to upload voice for WeCom: %w", err)
	}
	payload := struct {
		MediaID string `json:"media_id"`
	}{
		MediaID: mediaID,
	}
	return r.sendMessageToWeCom("voice", payload)
}

// SendVideoMessage 向企业微信机器人发送视频消息
// videoFilePath: 视频文件的本地路径
func (r *Robot) SendVideoMessage(videoFilePath string) error {
	mediaID, err := r.uploadMedia(videoFilePath, "video")
	if err != nil {
		return fmt.Errorf("failed to upload video for WeCom: %w", err)
	}
	payload := struct {
		MediaID string `json:"media_id"`
	}{
		MediaID: mediaID,
	}
	return r.sendMessageToWeCom("video", payload)
}

// SendFileMessage 向企业微信机器人发送文件消息
// filePath: 文件的本地路径
func (r *Robot) SendFileMessage(filePath string) error {
	mediaID, err := r.uploadMedia(filePath, "file")
	if err != nil {
		return fmt.Errorf("failed to upload file for WeCom: %w", err)
	}
	payload := struct {
		MediaID string `json:"media_id"`
	}{
		MediaID: mediaID,
	}
	return r.sendMessageToWeCom("file", payload)
}

// SendTextWithMentionMessage 向企业微信机器人发送带 @ 提醒的文本消息
// content: 文本消息内容
// mentionedList: 需要 @ 的成员 ID 列表，例如 ["userid1", "userid2"]
// mentionedMobileList: 需要 @ 的成员手机号列表，例如 ["13800000000", "@all"]
func (r *Robot) SendTextWithMentionMessage(content string, mentionedList []string, mentionedMobileList []string) error {
	payload := struct {
		Content             string   `json:"content"`
		MentionedList       []string `json:"mentioned_list,omitempty"`
		MentionedMobileList []string `json:"mentioned_mobile_list,omitempty"`
	}{
		Content:             content,
		MentionedList:       mentionedList,
		MentionedMobileList: mentionedMobileList,
	}
	return r.sendMessageToWeCom("text", payload)
}

// Article 定义图文消息中的文章结构
type Article struct {
	Title       string `json:"title"`       // 标题
	Description string `json:"description"` // 描述
	URL         string `json:"url"`         // 点击后跳转的链接
	PicURL      string `json:"picurl"`      // 图文消息的图片链接
}

// SendNewsMessage 向企业微信机器人发送图文消息
// articles: 文章列表，最多支持 8 条
func (r *Robot) SendNewsMessage(articles []Article) error {
	if len(articles) == 0 || len(articles) > 8 {
		return fmt.Errorf("news message must contain 1 to 8 articles")
	}
	payload := struct {
		Articles []Article `json:"articles"`
	}{
		Articles: articles,
	}
	return r.sendMessageToWeCom("news", payload)
}

// TemplateCard 定义模板卡片消息的结构
type TemplateCard struct {
	CardType          string        `json:"card_type"`                         // 卡片类型，固定为 "text_notice" 或 "news_notice"
	Source            interface{}   `json:"source,omitempty"`                  // 来源文案
	MainTitle         interface{}   `json:"main_title,omitempty"`              // 主标题
	QuoteArea         interface{}   `json:"quote_area,omitempty"`              // 引用区域
	SubTitleText      string        `json:"sub_title_text,omitempty"`          // 副标题
	HorizontalContent []interface{} `json:"horizontal_content_list,omitempty"` // 横向内容列表
	VerticalContent   []interface{} `json:"vertical_content_list,omitempty"`   // 纵向内容列表
	CardAction        interface{}   `json:"card_action,omitempty"`             // 整体点击跳转
	EmphasisContent   interface{}   `json:"emphasis_content,omitempty"`        // 关键数据区域
	ButtonSelection   interface{}   `json:"button_selection,omitempty"`        // 按钮选择
	ButtonList        []interface{} `json:"button_list,omitempty"`             // 按钮列表
}

// SendTemplateCardMessage 向企业微信机器人发送模板卡片消息
// card: 模板卡片内容
func (r *Robot) SendTemplateCardMessage(card TemplateCard) error {
	return r.sendMessageToWeCom("template_card", card)
}

// InteractiveCard 定义互动卡片消息的结构
type InteractiveCard struct {
	// 互动卡片字段，根据企业微信文档补充
	// 例如:
	// ActionMenu interface{} `json:"action_menu,omitempty"`
	// TaskID string `json:"task_id,omitempty"`
	// ...
}

// SendInteractiveCardMessage 向企业微信机器人发送互动卡片消息
// card: 互动卡片内容
func (r *Robot) SendInteractiveCardMessage(card InteractiveCard) error {
	return r.sendMessageToWeCom("interactive_card", card)
}
