# Dify API 完整文档

## 概述
Dify API 提供了完整的对话型应用开发接口，支持会话持久化、多模态输入、流式响应等功能。本文档详细说明各API的使用方法和参数说明。

---

## 基础信息

### 基础URL
```
https://api.dify.ai/v1
```

### 鉴权方式
所有API请求都需要在HTTP Header中包含API Key进行鉴权：

```http
Authorization: Bearer {API_KEY}
```

**安全建议**：  
强烈建议将API Key存储在后端服务中，避免在前端代码或客户端中直接暴露，以防止API Key泄露造成安全风险。

---

## 核心API

### 1. 对话消息API

#### 发送对话消息
`POST /chat-messages`

**功能描述**：  
创建并发送新的对话消息，支持文本、文件等多种输入形式，可选择流式或阻塞式响应模式。

**请求参数**：

| 参数名 | 类型 | 必填 | 描述 |
|--------|------|------|------|
| `query` | string | 是 | 用户输入/提问内容 |
| `inputs` | object | 否 | 应用定义的变量键值对，默认{} |
| `response_mode` | string | 是 | 响应模式：<br>- `streaming`：流式模式（推荐）<br>- `blocking`：阻塞模式 |
| `user` | string | 是 | 用户唯一标识 |
| `conversation_id` | string | 否 | 会话ID，用于继续之前的对话 |
| `files` | array | 否 | 文件列表，支持多种文件类型 |
| `auto_generate_name` | bool | 否 | 是否自动生成会话标题，默认true |

**文件对象结构**：
```json
{
  "type": "image|document|audio|video|custom",
  "transfer_method": "remote_url|local_file",
  "url": "string",  // remote_url时必填
  "upload_file_id": "string"  // local_file时必填
}
```

**响应示例**：
```json
{
  "event": "message",
  "task_id": "c3800678-a077-43df-a102-53f23ed20b88",
  "answer": "iPhone 13 Pro Max的规格是...",
  "metadata": {
    "usage": {
      "prompt_tokens": 1033,
      "total_price": "0.0012890",
      "currency": "USD"
    },
    "retriever_resources": [
      {
        "position": 1,
        "content": "详细的参考内容..."
      }
    ]
  }
}
```

**流式响应说明**：  
流式响应会通过Server-Sent Events(SSE)返回多个事件类型，包括：
- `message`: 文本内容块
- `message_file`: 文件展示事件
- `message_end`: 消息结束事件
- `error`: 错误事件

---

### 2. 文件上传API

#### 上传文件
`POST /files/upload`

**功能描述**：  
上传文件供后续对话使用，支持多种文件格式。

**请求格式**：  
`multipart/form-data`

**参数**：
- `file`: 要上传的文件
- `user`: 用户唯一标识

**响应字段说明**：
```json
{
  "id": "72fa9618-8f89-4a37-9b33-7e1178a24a67",
  "name": "example.png",
  "size": 1024,  // 文件大小(字节)
  "extension": "png",  // 文件扩展名
  "mime_type": "image/png",  // MIME类型
  "created_at": 1577836800  // 上传时间戳
}
```

**文件类型限制**：
- 图片：PNG, JPG, JPEG, WEBP, GIF
- 文档：TXT, PDF, DOCX, PPTX等
- 音频：MP3, WAV等
- 视频：MP4, MOV等

---

## 会话管理API

### 1. 获取会话列表
`GET /conversations`

**功能描述**：  
获取用户的会话历史列表，支持分页和排序。

**查询参数**：
| 参数 | 类型 | 说明 |
|------|------|------|
| `user` | string | 必填，用户唯一标识 |
| `last_id` | string | 分页游标，最后一条记录的ID |
| `limit` | int | 每页数量(1-100)，默认20 |
| `sort_by` | string | 排序字段：<br>`created_at`, `updated_at`等 |

**响应示例**：
```json
{
  "data": [{
    "id": "10799fb8-64f7-4296-bbf7-b42bfbe0ae54",
    "name": "iPhone规格咨询",
    "status": "normal",
    "created_at": 1679667915,
    "updated_at": 1679667915
  }],
  "has_more": false,
  "limit": 20
}
```

---

## 错误处理

### 常见错误代码
| 代码 | 类型 | 说明 |
|------|------|------|
| 400 | invalid_param | 参数格式错误 |
| 404 | not_found | 资源不存在 |
| 413 | file_too_large | 文件大小超过限制 |
| 429 | rate_limit | 请求过于频繁 |
| 500 | server_error | 服务器内部错误 |

**错误响应示例**：
```json
{
  "code": "invalid_param",
  "message": "Missing required parameter: user"
}
```

---

## 最佳实践

### 流式模式实现示例
```javascript
const eventSource = new EventSource('/chat-messages');

eventSource.onmessage = (event) => {
  const data = JSON.parse(event.data);
  switch(data.event) {
    case 'message':
      // 处理文本块
      break;
    case 'message_end':
      // 处理结束事件
      eventSource.close();
      break;
  }
};
```

### 安全建议
1. 始终在后端处理API Key
2. 对用户输入进行验证和清理
3. 设置合理的请求频率限制
