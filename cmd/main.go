package main

import (
	"bytes"         // 导入 bytes 包，用于处理字节缓冲区，例如构建 HTTP 请求体
	"encoding/json" // 导入 encoding/json 包，用于 JSON 数据的编解码
	"fmt"           // 导入 fmt 包，用于格式化字符串和错误信息
	"io"            // 导入 io 包，用于 IO 操作，例如读取响应体
	"log"           // 导入 log 包，用于日志输出
	"net/http"      // 导入 net/http 包，用于构建 HTTP 服务器和客户端
	"os"            // 导入 os 包，用于文件操作，例如设置日志输出到标准输出
	"time"          // 导入 time 包，用于处理时间相关操作，例如设置 HTTP 客户端超时时间

	"dify2wxbot/internal/config"  // 导入 internal/config 包，用于加载应用程序配置
	"dify2wxbot/internal/handler" // 导入 internal/handler 包，包含 WebhookHandler
	"dify2wxbot/internal/service" // 导入 internal/service 包，包含 DifyService 和 MessageConverter
	"dify2wxbot/internal/store"   // 导入 internal/store 包，包含 ConversationStore

	"github.com/robfig/cron/v3"        // 导入 cron 包，用于定时任务调度
	"gopkg.in/natefinch/lumberjack.v2" // 导入 lumberjack 包，用于日志文件轮转和管理
)

// Version 应用程序版本号
var Version = "v1.0.0" // 当前版本号

// main 函数是程序的入口点，负责初始化和启动各项服务
func main() {
	// 打印应用程序版本信息
	fmt.Printf("Dify2WxBot 应用程序版本: %s\n", Version)

	// 加载应用程序配置
	cfg, err := config.LoadConfig()
	if err != nil {
		// 如果配置加载失败，则记录致命错误并退出程序
		log.Fatalf("配置加载失败: %v", err)
	}

	// 根据配置设置日志输出目标
	if cfg.LogToFile {
		// 如果配置中启用了日志文件输出，则配置 lumberjack 日志轮转器
		logOutput := &lumberjack.Logger{
			Filename:   cfg.LogFilePath,     // 日志文件路径，例如 "logs/app.log"
			MaxSize:    cfg.LogMaxSizeBytes, // 单个日志文件的最大大小（MB），达到此大小后会进行切割
			MaxBackups: cfg.LogMaxBackups,   // 保留旧日志文件的最大个数，超出此数量的旧文件会被删除
			MaxAge:     cfg.LogMaxAgeDays,   // 保留旧日志文件的最大天数，超出此天数的旧文件会被删除
			Compress:   cfg.LogCompress,     // 是否压缩旧的日志文件（gzip 格式）
		}
		log.SetOutput(logOutput)                             // 将标准日志库的输出重定向到 lumberjack
		log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile) // 设置日志格式，包含日期、时间、文件名和行号
		log.Printf("日志已重定向到文件: %s (最大大小: %dMB, 最大备份: %d, 最大天数: %d, 压缩: %t)",
			cfg.LogFilePath, cfg.LogMaxSizeBytes, cfg.LogMaxBackups, cfg.LogMaxAgeDays, cfg.LogCompress)
	} else {
		// 如果未启用日志文件输出，则默认将日志输出到标准输出 (控制台)
		log.SetOutput(os.Stdout)                             // 将日志输出设置为标准输出
		log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile) // 设置日志格式，包含日期、时间、文件名和行号
	}

	// 创建 DifyService 实例，用于与 Dify AI 服务的 API 进行交互
	difyService := service.NewDifyService(cfg)

	// 创建 MessageConverter 实例，负责将 Dify 的回复消息格式化并发送到企业微信群机器人
	messageConverter := service.NewMessageConverter(cfg, difyService)

	// 创建 ConversationStore 实例，用于管理用户与 Dify 之间的对话 ID，以维持上下文
	conversationStore := store.NewInMemoryConversationStore()

	// 创建 WebhookHandler 实例，用于处理所有传入的 HTTP Webhook 请求
	webhookHandler := handler.NewWebhookHandler(messageConverter, conversationStore, cfg)

	// 注册 Webhook 路由，将所有 "/webhook" 路径的请求路由到 webhookHandler 的 HandleWebhook 方法
	http.HandleFunc("/webhook", webhookHandler.HandleWebhook)

	// 创建一个可重用的 HTTP 客户端实例，用于发送定时任务请求
	httpClient := &http.Client{
		Timeout: 10 * time.Second, // 设置 HTTP 请求的超时时间为 10 秒，防止长时间阻塞
	}

	// 初始化一个全局的 Cron 调度器实例，用于管理和执行定时任务
	c := cron.New()

	// 启动定时任务（如果配置中启用了定时器）
	// 遍历配置文件中定义的所有定时器配置
	for i, schedulerCfg := range cfg.Schedulers {
		// 检查当前定时器是否被启用
		if !schedulerCfg.Enable {
			log.Printf("定时器 %d: 未启用，跳过配置和启动。", i)
			continue // 跳过当前定时器的处理
		}

		// 为每个定时器定义一个独立的执行函数（闭包）
		// 捕获当前 schedulerCfg 的值，避免在循环中因闭包引用外部变量导致的问题
		currentSchedulerCfg := schedulerCfg
		taskName := fmt.Sprintf("定时器 %d", i) // 为当前定时任务生成一个唯一的名称，用于日志记录
		scheduleTask := func() {
			log.Printf("%s 触发，正在调用目标 URL: %s", taskName, currentSchedulerCfg.TargetURL)
			// 构建发送到目标 URL 的请求体，包含默认消息和用户标识
			requestBody := map[string]string{
				"message": currentSchedulerCfg.DefaultMessage,
				"user":    fmt.Sprintf("scheduler_bot_%d", i), // 定时任务的默认用户标识，带序号区分，便于追踪
			}
			// 将请求体编码为 JSON 格式
			jsonBody, err := json.Marshal(requestBody)
			if err != nil {
				log.Printf("%s：JSON 编码请求体失败: %v", taskName, err)
				return // 如果编码失败，则终止当前任务的执行
			}

			// 创建一个新的 HTTP POST 请求
			req, err := http.NewRequest(http.MethodPost, currentSchedulerCfg.TargetURL, bytes.NewBuffer(jsonBody))
			if err != nil {
				log.Printf("%s：创建 HTTP 请求失败: %v", taskName, err)
				return // 如果请求创建失败，则终止当前任务的执行
			}
			req.Header.Set("Content-Type", "application/json") // 设置请求头为 JSON 格式

			// 如果应用程序配置中启用了认证功能，则添加 Authorization 头
			if cfg.EnableAuth { // 注意：这里的认证 Token 是全局的，所有定时任务共享
				req.Header.Set("Authorization", "Bearer "+cfg.AuthToken)
			}

			// 使用预先创建的可重用 HTTP 客户端发送请求
			resp, err := httpClient.Do(req)
			if err != nil {
				log.Printf("%s：发送 HTTP 请求失败: %v", taskName, err)
				return // 如果请求发送失败，则终止当前任务的执行
			}
			defer resp.Body.Close() // 确保在函数返回前关闭响应体，释放资源

			// 检查 HTTP 响应状态码是否为 200 OK
			if resp.StatusCode != http.StatusOK {
				// 如果状态码不是 200，则读取响应体并记录详细错误日志
				bodyBytes, _ := io.ReadAll(resp.Body) // 尝试读取响应体内容
				log.Printf("%s：HTTP 请求返回非 200 状态码: %d, 响应体: %s", taskName, resp.StatusCode, string(bodyBytes))
			} else {
				// 如果状态码是 200 OK，则记录请求成功日志
				log.Printf("%s：HTTP 请求成功。", taskName)
			}
		}

		// 优先使用 Cron 表达式进行调度（如果配置了 CronSpec）
		if currentSchedulerCfg.CronSpec != "" {
			// 将定时任务添加到 Cron 调度器
			_, err := c.AddFunc(currentSchedulerCfg.CronSpec, scheduleTask)
			if err != nil {
				// 如果添加 Cron 表达式失败，则记录致命错误并退出程序，因为这通常是配置问题
				log.Fatalf("%s：添加 Cron 表达式失败: %v", taskName, err)
			}
			log.Printf("%s 已启动，将使用 Cron 表达式: '%s' 定期调用 %s", taskName, currentSchedulerCfg.CronSpec, currentSchedulerCfg.TargetURL)
		} else {
			// 如果没有配置 Cron 表达式，则回退到旧的周期性调度方式
			// 将周期性调度（间隔时间和单位）转换为 Cron 表达式，例如 "@every 1m"
			var cronSpec string
			switch currentSchedulerCfg.Unit {
			case "second": // 单位为秒
				cronSpec = fmt.Sprintf("@every %ds", currentSchedulerCfg.Interval)
			case "minute": // 单位为分钟
				cronSpec = fmt.Sprintf("@every %dm", currentSchedulerCfg.Interval)
			case "hour": // 单位为小时
				cronSpec = fmt.Sprintf("@every %dh", currentSchedulerCfg.Interval)
			default: // 未知的单位
				log.Printf("%s：检测到未知的定时任务单位: '%s'，此定时任务将不会启动。", taskName, currentSchedulerCfg.Unit)
				continue // 跳过当前定时任务的配置
			}

			// 检查定时任务间隔时间是否有效
			if currentSchedulerCfg.Interval <= 0 {
				log.Printf("%s：定时任务间隔时间必须大于 0，此定时任务将不会启动。", taskName)
				continue // 跳过当前定时任务的配置
			}

			// 将转换后的周期性 Cron 表达式添加到调度器
			_, err := c.AddFunc(cronSpec, scheduleTask)
			if err != nil {
				// 如果添加周期性 Cron 表达式失败，则记录致命错误并退出程序
				log.Fatalf("%s：添加周期性 Cron 表达式失败: %v", taskName, err)
			}
			log.Printf("%s 已启动，将每隔 %d %s 调用一次 %s (转换为 Cron 表达式: '%s')",
				taskName, currentSchedulerCfg.Interval, currentSchedulerCfg.Unit, currentSchedulerCfg.TargetURL, cronSpec)
		}
	}

	// 启动 Cron 调度器 (在所有定时任务添加完毕后统一启动，使其开始执行)
	c.Start()

	// 启动 HTTP 服务器，监听指定端口
	port := ":8080" // 服务器监听的端口号
	log.Printf("服务器正在端口 %s 上启动并监听传入请求...", port)
	// 启动 HTTP 服务器，如果启动失败（例如端口被占用），则记录致命错误并退出
	err = http.ListenAndServe(port, nil)
	if err != nil {
		log.Fatalf("服务器启动失败: %v", err)
	}
	// 为了确保程序持续运行，如果 HTTP 服务器没有阻塞，可以添加一个阻塞语句
	// 例如：select {}
	// 但 http.ListenAndServe 理论上是阻塞的，如果它立即返回，说明有错误发生并被 log.Fatalf 捕获。
	// 如果程序仍然立即退出且没有日志，可能是日志配置问题或环境问题。
}
