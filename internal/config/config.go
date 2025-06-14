package config

import (
	"fmt"           // 导入 fmt 包，用于格式化字符串和错误信息
	"os"            // 导入 os 包，用于文件操作和环境变量读取
	"path/filepath" // 导入 filepath 包，用于处理文件路径
	"strconv"       // 导入 strconv 包，用于字符串和基本数据类型之间的转换

	"gopkg.in/yaml.v2" // 导入 yaml.v2 包，用于 YAML 文件的编解码
)

// DifyConfig 结构体定义了 Dify API 的配置
type DifyConfig struct {
	APIKey        string `yaml:"api_key"`        // Dify API 密钥，用于认证 Dify API 请求
	BaseURL       string `yaml:"base_url"`       // Dify API 基础 URL，例如 "https://api.dify.ai"
	BotType       string `yaml:"bot_type"`       // Dify 应用类型，可以是 "chat", "completion", "workflow"
	WorkflowID    string `yaml:"workflow_id"`    // Dify Workflow 应用的 ID，仅当 BotType 为 "workflow" 时需要
	DefaultPrompt string `yaml:"default_prompt"` // 默认提示词，当用户消息为空时使用，或用于定时任务的默认输入
}

// WeComConfig 结构体定义了企业微信机器人的配置
type WeComConfig struct {
	WebhookURL string `yaml:"webhook_url"` // 企业微信机器人 Webhook URL，用于发送消息到企业微信群
}

// SchedulerConfig 结构体定义了定时任务的配置
type SchedulerConfig struct {
	Enable         bool   `yaml:"enable"`          // 是否启用当前定时任务 (true: 启用, false: 禁用)
	CronSpec       string `yaml:"cron_spec"`       // Cron 表达式，用于更灵活的定时调度，例如 "0 0 * * *" 表示每天午夜执行
	Interval       int    `yaml:"interval"`        // 定时任务间隔时间，当 CronSpec 为空时生效，表示每隔多少单位时间执行一次
	Unit           string `yaml:"unit"`            // 时间单位，当 CronSpec 为空时生效，可以是 "second", "minute", "hour"
	TargetURL      string `yaml:"target_url"`      // 定时调用的目标 URL，通常是本服务的 Webhook 地址
	DefaultMessage string `yaml:"default_message"` // 定时调用时发送的默认消息内容
}

// AppConfig 结构体定义了整个应用程序的配置
type AppConfig struct {
	Dify            DifyConfig        `yaml:"dify"`             // Dify 配置部分，包含 Dify API 相关的设置
	WeCom           WeComConfig       `yaml:"wecom"`            // WeCom (企业微信) 配置部分，包含企业微信机器人相关的设置
	AuthToken       string            `yaml:"auth_token"`       // 用于 Webhook 认证的 Token，客户端请求时需在 Authorization 头中携带
	EnableAuth      bool              `yaml:"enable_auth"`      // 是否开启认证 Token 功能，如果为 true，则所有 Webhook 请求都需要认证
	Schedulers      []SchedulerConfig `yaml:"schedulers"`       // 定时任务配置列表部分，支持配置多个独立的定时器
	LogToFile       bool              `yaml:"log_to_file"`      // 是否将日志输出到文件，如果为 true，日志将写入到指定文件
	LogFilePath     string            `yaml:"log_file_path"`    // 日志文件路径，当 log_to_file 为 true 时生效，例如 "logs/app.log"
	LogMaxSizeBytes int               `yaml:"log_max_size_mb"`  // 日志文件最大大小 (MB)，达到此大小后会进行切割，防止单个日志文件过大
	LogMaxBackups   int               `yaml:"log_max_backups"`  // 日志文件最大备份数量，超出此数量的旧文件会被删除
	LogMaxAgeDays   int               `yaml:"log_max_age_days"` // 日志文件最大保留天数，超出此天数的旧文件会被删除
	LogCompress     bool              `yaml:"log_compress"`     // 是否压缩旧的日志文件（gzip 格式），以节省存储空间
}

// Validate 方法用于验证 AppConfig 结构体中的必要配置项是否已设置
// 如果有任何必要配置项缺失，将返回一个错误
func (c *AppConfig) Validate() error {
	// 检查 Dify API Key 是否为空，这是与 Dify 交互的必备条件
	if c.Dify.APIKey == "" {
		return fmt.Errorf("dify API Key 未配置")
	}
	// 检查 Dify Base URL 是否为空，这是 Dify API 的访问地址
	if c.Dify.BaseURL == "" {
		return fmt.Errorf("dify Base URL 未配置")
	}
	// 检查企业微信 Webhook URL 是否为空，这是发送消息到企业微信的必要条件
	if c.WeCom.WebhookURL == "" {
		return fmt.Errorf("企业微信 Webhook URL 未配置")
	}
	// 如果配置中开启了认证功能，则检查 Auth Token 是否已配置
	if c.EnableAuth && c.AuthToken == "" {
		return fmt.Errorf("认证 Token 已开启但未配置")
	}
	return nil // 所有必要配置都已设置，返回 nil 表示验证成功
}

// LoadConfig 函数用于加载应用程序配置
// 它首先尝试从名为 "config/config.yaml" 的 YAML 文件加载配置。
// 如果 YAML 文件不存在或加载失败，它将回退到从环境变量加载配置。
// 注意：从环境变量加载时，只支持单个定时器配置。
func LoadConfig() (*AppConfig, error) {
	var cfg AppConfig // 声明一个 AppConfig 变量用于存储加载的配置
	var err error     // 声明一个 error 变量用于捕获可能发生的错误

	// 尝试从 YAML 文件加载配置
	yamlConfigPath := filepath.Join("internal", "config", "config.yaml") // 构建 YAML 配置文件的完整路径
	// 检查 YAML 配置文件是否存在
	if _, fileErr := os.Stat(yamlConfigPath); fileErr == nil { // os.Stat 返回文件信息，如果文件存在则 fileErr 为 nil
		data, readErr := os.ReadFile(yamlConfigPath) // 读取 YAML 文件内容到字节切片
		if readErr != nil {
			return nil, fmt.Errorf("读取 YAML 配置失败: %v", readErr) // 如果读取失败，返回错误
		}

		// 替换 YAML 内容中的环境变量占位符（例如 ${ENV_VAR}），允许配置中引用环境变量
		configContent := os.ExpandEnv(string(data))

		// 将 YAML 内容解析（反序列化）到 AppConfig 结构体
		if unmarshalErr := yaml.Unmarshal([]byte(configContent), &cfg); unmarshalErr != nil {
			return nil, fmt.Errorf("解析 YAML 配置失败: %v", unmarshalErr) // 如果解析失败，返回错误
		}
	} else if os.IsNotExist(fileErr) { // 如果 YAML 文件不存在 (os.IsNotExist 判断是否为文件不存在错误)
		// 回退到从环境变量加载配置 (这种方式主要用于部署在容器环境等场景)
		// 注意：从环境变量加载时，Schedulers 列表只支持配置一个定时器
		cfg = AppConfig{
			Dify: DifyConfig{ // Dify API 配置部分
				APIKey:        os.Getenv("DIFY_API_KEY"),        // 从环境变量 DIFY_API_KEY 获取 Dify API 密钥
				BaseURL:       os.Getenv("DIFY_BASE_URL"),       // 从环境变量 DIFY_BASE_URL 获取 Dify API 基础 URL
				BotType:       os.Getenv("DIFY_BOT_TYPE"),       // 从环境变量 DIFY_BOT_TYPE 获取 Dify 应用类型
				WorkflowID:    os.Getenv("DIFY_WORKFLOW_ID"),    // 从环境变量 DIFY_WORKFLOW_ID 获取 Dify Workflow ID
				DefaultPrompt: os.Getenv("DIFY_DEFAULT_PROMPT"), // 从环境变量 DIFY_DEFAULT_PROMPT 获取默认提示词
			},
			WeCom: WeComConfig{ // 企业微信机器人配置部分
				WebhookURL: os.Getenv("WECHAT_WEBHOOK_URL"), // 从环境变量 WECHAT_WEBHOOK_URL 获取企业微信 Webhook URL
			},
			AuthToken:       os.Getenv("AUTH_TOKEN"),                     // 从环境变量 AUTH_TOKEN 获取认证 Token
			EnableAuth:      os.Getenv("ENABLE_AUTH") == "true",          // 从环境变量 ENABLE_AUTH 获取是否开启认证功能
			LogToFile:       os.Getenv("LOG_TO_FILE") == "true",          // 从环境变量 LOG_TO_FILE 获取是否将日志输出到文件
			LogFilePath:     os.Getenv("LOG_FILE_PATH"),                  // 从环境变量 LOG_FILE_PATH 获取日志文件路径
			LogMaxSizeBytes: parseInt(os.Getenv("LOG_MAX_SIZE_MB"), 100), // 从环境变量 LOG_MAX_SIZE_MB 获取日志文件最大大小，并提供默认值 100MB
			LogMaxBackups:   parseInt(os.Getenv("LOG_MAX_BACKUPS"), 0),   // 从环境变量 LOG_MAX_BACKUPS 获取日志文件最大备份数量，并提供默认值 0
			LogMaxAgeDays:   parseInt(os.Getenv("LOG_MAX_AGE_DAYS"), 0),  // 从环境变量 LOG_MAX_AGE_DAYS 获取日志文件最大保留天数，并提供默认值 0
			LogCompress:     os.Getenv("LOG_COMPRESS") == "true",         // 从环境变量 LOG_COMPRESS 获取是否压缩旧日志文件
			Schedulers: []SchedulerConfig{ // 定时任务配置列表，从环境变量加载时只支持一个定时器
				{
					Enable:         os.Getenv("SCHEDULER_ENABLE") == "true",      // 从环境变量 SCHEDULER_ENABLE 获取是否启用定时任务
					CronSpec:       os.Getenv("SCHEDULER_CRON_SPEC"),             // 从环境变量 SCHEDULER_CRON_SPEC 获取 Cron 表达式
					Interval:       parseInt(os.Getenv("SCHEDULER_INTERVAL"), 0), // 从环境变量 SCHEDULER_INTERVAL 获取间隔时间，并提供默认值 0
					Unit:           os.Getenv("SCHEDULER_UNIT"),                  // 从环境变量 SCHEDULER_UNIT 获取时间单位
					TargetURL:      os.Getenv("SCHEDULER_TARGET_URL"),            // 从环境变量 SCHEDULER_TARGET_URL 获取目标 URL
					DefaultMessage: os.Getenv("SCHEDULER_DEFAULT_MESSAGE"),       // 从环境变量 SCHEDULER_DEFAULT_MESSAGE 获取默认消息
				},
			},
		}
	} else { // 如果是其他文件系统错误（例如权限问题），则返回错误
		return nil, fmt.Errorf("检查 YAML 配置文件失败: %v", fileErr)
	}

	// 统一验证加载到的配置，无论从 YAML 还是环境变量加载，都进行验证
	if err = cfg.Validate(); err != nil {
		return nil, fmt.Errorf("配置验证失败: %v", err) // 如果验证失败，返回错误
	}

	return &cfg, nil // 返回加载并验证成功的配置指针
}

// parseInt 辅助函数，用于将字符串转换为整数
// s: 待转换的字符串
// defaultValue: 如果字符串为空或转换失败时返回的默认值
func parseInt(s string, defaultValue int) int {
	if s == "" { // 如果输入字符串为空，直接返回默认值
		return defaultValue
	}
	i, err := strconv.Atoi(s) // 尝试将字符串转换为整数
	if err != nil {
		// 如果转换失败，返回默认值。
		// 注意：这里没有使用 log 包进行日志记录，因为 config 包不应该直接依赖 log 包，
		// 以避免循环依赖或不必要的耦合。调用方应该处理此函数的返回值。
		return defaultValue
	}
	return i // 返回成功转换后的整数
}
