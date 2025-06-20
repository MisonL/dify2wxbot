<div align="center">

# Dify to WeCom Bot

![Go Version](https://img.shields.io/badge/Go-1.21%2B-00ADD8?style=for-the-badge&logo=go) ![License](https://img.shields.io/badge/License-MIT-yellow.svg?style=for-the-badge) ![Version](https://img.shields.io/badge/Version-v1.0.0-blue.svg?style=for-the-badge)

</div>

一个 Go 语言服务，用于调用 Dify API 获取 AI 响应，并将响应消息转发到企业微信(WeCom)机器人。

## ✨ 功能特性

-   **灵活的配置管理**: 支持通过 `config.yaml` 文件或环境变量加载配置，并支持环境变量引用。
-   **增强的日志管理**: 集成 `lumberjack` 库，实现日志文件的自动切割、备份、按天保留和压缩。
-   **统一的定时任务调度**: 程序支持配置多个独立的定时任务，每个任务可以通过标准的 Cron 表达式（如 `0 8 * * *` 表示每天早上 8 点）或简单的周期性间隔（如每 5 分钟）进行灵活调度。定时任务触发时，会自动向指定的目标 URL 发送 Webhook 请求，实现自动化消息推送或业务触发。
-   **Dify API 集成**: 支持调用 Dify 的 `chat-messages`、`completion-messages` 和 `workflows/run` API 获取 AI 生成的回复或执行工作流。
-   **Dify 文件上传**: 支持将文件上传到 Dify，并在聊天消息中引用。
-   **企业微信消息转发**: 支持将 Dify 的 AI 回复发送到企业微信群机器人，支持发送文本、Markdown (v1 和 v2)、图片、语音、视频、文件、带 @ 提醒的文本、图文、模板卡片和互动卡片消息，并处理消息长度截断。
-   **Webhook 接收与处理**: 实现 HTTP 服务器接收 Webhook 请求，支持 JSON 和 `multipart/form-data` (含文件上传)，能够自动识别并处理用户上传的文件。
-   **请求认证**: 可选的 Webhook 请求认证功能，通过 `Authorization` 头进行验证。
-   **健壮的错误处理**: 包含 Dify API 请求重试机制、文件操作错误处理、详细的错误日志，并针对企业微信 API 频率限制（错误码 45009）提供日志警告。
-   **对话上下文管理**: 智能管理用户与 Dify 之间的对话上下文。程序优先使用请求中提供的 `conversation_id`；如果未提供，则尝试从本地存储中获取；如果本地存储中也不存在，则将 `conversation_id` 留空，让 Dify 服务自动创建新的会话。
-   **模块化设计**: 清晰的服务层和处理层分离，易于扩展和维护。

## 🚀 快速开始

### ⚙️ 环境要求

-   Go 1.21+ (或更高版本)
-   Dify API Key
-   企业微信机器人 Webhook URL (包含 `key` 参数)

### 配置

应用程序会按以下顺序加载配置：

1.  **`config/config.yaml` 文件**: 如果 `config/config.yaml` 文件存在，程序将尝试从中加载配置。该文件支持使用 `${ENV_VAR}` 语法引用环境变量。
2.  **环境变量**: 如果 `config.yaml` 文件不存在或加载失败，程序将从环境变量中加载配置。

**推荐配置方式 (二选一)**:

**方式一：使用 `config/config.yaml` (推荐)**

在项目根目录下创建或修改 `config/config.yaml` 文件，内容如下：

```yaml
# 应用配置示例
dify:
  api_key: ${DIFY_API_KEY}  # 必须通过环境变量设置，或直接在此处填写
  base_url: "https://api.dify.ai" # Dify API 基础 URL，必须使用 HTTPS 协议
  bot_type: "chat" # Dify 应用类型: "chat", "completion", "workflow"
  workflow_id: "" # 仅当 bot_type 为 "workflow" 时需要填写
  default_prompt: "你好" # 当用户消息为空时，发送给 Dify 的默认提示词

wecom:
  webhook_url: ${WECHAT_WEBHOOK_URL} # 完整的企业微信机器人 Webhook URL (包含 key 参数)，必须通过环境变量设置，或直接在此处填写

auth_token: ${AUTH_TOKEN} # 可选：用于 Webhook 认证的 Token
enable_auth: false # 是否开启认证功能，默认为 false

log_to_file: false # 是否将日志输出到文件，默认为 false
log_file_path: "logs/app.log" # 日志文件路径
log_max_size_mb: 100 # 日志文件最大大小 (MB)
log_max_backups: 5 # 日志文件最大备份数量
log_max_age_days: 30 # 日志文件最大保留天数
log_compress: true # 是否压缩旧的日志文件

schedulers: # 定时任务配置列表，支持配置多个独立的定时器。
  - enable: false # `true` 启用此定时器，`false` 禁用。
    cron_spec: "0 8 * * *" # 可选。标准的 Cron 表达式，例如 "0 8 * * *" 表示每天早上 8 点执行。如果设置了此项，`interval` 和 `unit` 将被忽略。
    interval: 0 # 可选。当 `cron_spec` 为空时生效，表示任务执行的间隔时间（整数）。
    unit: "minute" # 可选。当 `cron_spec` 为空时生效，表示 `interval` 的时间单位，可选值包括 "second", "minute", "hour"。
    target_url: "http://localhost:7860/webhook" # 必填。定时任务触发时，程序将向此 URL 发送 POST 请求。通常指向本服务的 `/webhook` 接口。
    default_message: "早上好，今天有什么新消息？" # 必填。定时任务发送 Webhook 请求时，请求体中 `message` 字段的默认内容。
  # 您可以根据需要添加更多定时器配置，每个定时器都是一个独立的 `-` 项。
  # 例如：
  # - enable: true
  #   cron_spec: "@every 1h" # 每小时执行一次
  #   target_url: "http://localhost:7860/webhook"
  #   default_message: "每小时提醒：请检查最新通知。"
  # - enable: true
  #   interval: 30 # 每 30 秒执行一次
  #   unit: "second"
  #   target_url: "http://localhost:7860/webhook"
  #   default_message: "快速检查"
```

然后，确保设置以下环境变量（根据您的配置方式和需求）：

```bash
export DIFY_API_KEY="your_dify_api_key"
export DIFY_BASE_URL="https://api.dify.ai" # 例如：https://api.dify.ai 或您的自托管地址
export DIFY_BOT_TYPE="chat" # 例如："chat", "completion", "workflow"
export DIFY_WORKFLOW_ID="" # 如果使用 workflow 类型，填写您的 workflow ID
export DIFY_DEFAULT_PROMPT="你好"

export WECHAT_WEBHOOK_URL="your_wechat_webhook_url"

export AUTH_TOKEN="your_auth_token" # 如果 enable_auth 为 true，则需要设置
export ENABLE_AUTH="false" # "true" 或 "false"

export LOG_TO_FILE="false" # "true" 或 "false"
export LOG_FILE_PATH="logs/app.log"
export LOG_MAX_SIZE_MB="100"
export LOG_MAX_BACKUPS="5"
export LOG_MAX_AGE_DAYS="30"
export LOG_COMPRESS="true" # "true" 或 "false"

# 如果只配置一个定时器，可以使用以下环境变量
export SCHEDULER_ENABLE="false" # "true" 或 "false"
export SCHEDULER_CRON_SPEC="0 8 * * *"
export SCHEDULER_INTERVAL="0"
export SCHEDULER_UNIT="minute"
export SCHEDULER_TARGET_URL="http://localhost:7860/webhook"
export SCHEDULER_DEFAULT_MESSAGE="早上好，今天有什么新消息？"
```

**方式二：仅使用环境变量**

直接设置上述所有相关的环境变量。

### 📦 安装依赖

```bash
go mod tidy
```

### ▶️ 运行

```bash
go run cmd/main.go
```

## ⚙️ 编译并运行二进制文件

如果您不想使用 Docker，也可以直接编译 Go 应用程序并运行。

### 步骤

1.  **编译应用程序**:
    在项目根目录下执行以下命令来编译应用程序：
    ```bash
    go build -o dify2wxbot ./cmd/main.go
    ```
    这会在当前目录下生成一个名为 `dify2wxbot` 的可执行文件。

2.  **配置环境变量 (如果需要)**:
    如果您的配置依赖于环境变量（例如 `DIFY_API_KEY`, `WECHAT_WEBHOOK_URL` 等），请在运行二进制文件之前设置它们。例如：
    ```bash
    export DIFY_API_KEY="your_dify_api_key"
    export WECHAT_WEBHOOK_URL="your_wechat_webhook_url"
    # ... 其他环境变量
    ```

3.  **运行应用程序**:
    直接运行编译好的二进制文件：
    ```bash
    ./dify2wxbot
    ```
    应用程序将启动并监听配置的端口（默认为 7860）。

## 📦 本地 Docker 部署

您也可以选择在本地构建 Docker 镜像并运行。

### 步骤

1.  **构建 Docker 镜像**:
    在项目根目录下执行以下命令来构建 Docker 镜像：
    ```bash
    docker build -t dify2wxbot .
    ```
    这会根据 `Dockerfile` 构建一个名为 `dify2wxbot` 的镜像。

2.  **运行 Docker 容器**:
    构建完成后，您可以运行以下命令来启动容器：
    ```bash
    docker run -d -p 7860:7860 --name dify2wxbot_container dify2wxbot
    ```
    *   `-d`：在后台运行容器。
    *   `-p 7860:7860`：将主机的 7860 端口映射到容器的 7860 端口。
    *   `--name dify2wxbot_container`：为容器指定一个名称。
    *   `dify2wxbot`：指定要运行的镜像名称。

    **配置环境变量**:
    如果您的配置依赖于环境变量（例如 `DIFY_API_KEY`, `WECHAT_WEBHOOK_URL` 等），您可以在 `docker run` 命令中使用 `-e` 参数来设置它们：
    ```bash
    docker run -d -p 7860:7860 \
      -e DIFY_API_KEY="your_dify_api_key" \
      -e WECHAT_WEBHOOK_URL="your_wechat_webhook_url" \
      --name dify2wxbot_container dify2wxbot
    ```
    请根据您的实际配置需求添加或修改环境变量。

3.  **查看日志 (可选)**:
    要查看容器的运行日志，可以使用：
    ```bash
    docker logs dify2wxbot_container
    ```

4.  **停止并移除容器 (可选)**:
    要停止正在运行的容器：
    ```bash
    docker stop dify2wxbot_container
    ```
    要移除容器：
    ```bash
    docker rm dify2wxbot_container
    ```

## 🚀 部署到 Hugging Face Spaces

您可以轻松地将此服务部署到 Hugging Face Spaces。

### 步骤

1.  **创建新的 Space**:
    *   访问 [Hugging Face Spaces](https://huggingface.co/spaces)。
    *   点击 "Create new Space" (创建新 Space)。
    *   为您的 Space 命名，并选择 "Docker" 作为 SDK。
    *   选择一个合适的硬件配置（例如，免费的 CPU Basic 即可满足基本需求）。
    *   点击 "Create Space" (创建 Space)。

2.  **上传代码**:
    *   创建 Space 后，您可以通过 Git 克隆 Space 仓库到本地，然后将本项目的所有文件（包括 `Dockerfile`、`go.mod`、`cmd/`、`internal/`、`config/` 等）复制到该仓库中。
    *   提交并推送您的代码到 Space 仓库。

3.  **配置环境变量**:
    *   在您的 Hugging Face Space 页面，导航到 "Settings" (设置) 选项卡。
    *   在 "Repository secrets" (仓库密钥) 部分，添加您的 Dify API Key 和企业微信 Webhook URL 等环境变量。这些变量名应与 `config.yaml` 或您在本地设置的环境变量名称一致（例如 `DIFY_API_KEY`, `WECHAT_WEBHOOK_URL`, `AUTH_TOKEN` 等）。
    *   如果您的 `config.yaml` 中使用了 `${ENV_VAR}` 引用，确保在 Space 中设置了对应的环境变量。

4.  **启动应用**:
    *   Hugging Face Spaces 会自动检测 `Dockerfile` 并构建您的应用程序。
    *   构建完成后，应用程序将自动启动并监听 7860 端口。
    *   您可以在 Space 页面上查看日志以监控应用程序的运行状态。

### 💡 使用指南

程序运行后，会启动一个 HTTP 服务器监听 `/webhook` 路径，并根据配置启动定时任务。

**通过 Webhook 调用**:

您可以向 `http://localhost:7860/webhook` 发送 POST 请求来与 Dify 机器人交互。

**JSON 请求示例**:

```bash
curl -X POST http://localhost:7860/webhook \
-H "Content-Type: application/json" \
-d '{
    "message": "你好，Dify机器人！",
    "user": "test_user_123",
    "conversation_id": "optional_conversation_id"
}'
```

如果启用了认证：

```bash
curl -X POST http://localhost:7860/webhook \
-H "Content-Type: application/json" \
-H "Authorization: Bearer your_auth_token" \
-d '{
    "message": "你好，Dify机器人！",
    "user": "test_user_123"
}'
```

**文件上传请求示例 (multipart/form-data)**:

```bash
curl -X POST http://localhost:7860/webhook \
-H "Content-Type: multipart/form-data" \
-F "message=这是一张图片" \
-F "user=test_user_file" \
-F "file=@/path/to/your/image.png" # 替换为您的图片路径
```

**定时任务**:

如果配置中启用了定时任务，程序将按照您在 `config.yaml` 中定义的 Cron 表达式或周期性间隔（秒、分钟、小时）自动向 `target_url` 发送 Webhook 请求。这使得您可以轻松实现定时提醒、定期数据同步或自动化报告等功能。请参考 [配置](#配置) 部分了解详细的定时任务配置方法。


## 🧑‍💻 开发

### 项目结构

```bash
.
├── .gitignore      # Git 忽略文件配置
├── Dockerfile      # Docker 镜像构建文件
├── go.mod          # Go 模块定义文件
├── LICENSE         # 项目许可证
├── LICENSE_zh-CN   # 项目许可证 (中文)
├── README.md       # 项目说明文件 (自身)
├── cmd/            # 主程序入口，包含 main 函数
│   └── main.go
├── docs/           # 文档目录
│   ├── dify_api_documentation_full.md # Dify API 完整文档
│   └── wecom_robot_config.md # 企业微信机器人配置文档
└── internal/       # 内部实现，不应被外部包直接引用
    ├── config/     # 应用程序配置相关文件
    │   ├── config.go   # 配置结构体和加载逻辑
    │   └── config.yaml # 配置文件示例
    ├── handler/    # HTTP 请求处理器，例如 Webhook 处理
    │   └── webhook.go
    ├── service/    # 业务逻辑服务层
    │   ├── converter.go # 消息转换和发送服务
    │   └── dify_service.go # Dify API 交互服务
    └── store/      # 数据存储层
        └── conversation_store.go # 对话上下文存储
```

## 关于

**作者**: Mison
**邮箱**: 1360962086@qq.com
**许可证**: MIT
