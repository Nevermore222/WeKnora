# ASP 知识库 Agent 调用说明

## 1. 基本信息

```text
Base URL: http://192.168.9.177:8080/api/v1
认证：X-API-Key 请求头
```

| Agent | Agent ID | 绑定知识库 |
|---|---|---|
| ASP-JP1 专家 | `585d7108-b55b-4be5-86c1-ba53a200c86b` | `ASP-JP1` |
| ASP-BSP-RM 账票专家 | `dd87a237-9b65-4fe2-a7d4-5a4dbf995c8a` | `ASP-BSP-RM账票KnowDB` |

不要在代码、日志或前端源码中硬编码 API Key。以下示例使用占位符：

```bash
export XELORA_URL="http://192.168.9.177:8080/api/v1"
export XELORA_API_KEY="sk-替换为实际APIKey"
```

## 2. 创建会话

每次问答先创建会话，保存返回的 `data.id`：

```bash
curl -sS -X POST "$XELORA_URL/sessions" \
  -H "X-API-Key: $XELORA_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"title":"ASP 知识库查询"}'
```

响应示例：

```json
{"success":true,"data":{"id":"<session-id>"}}
```

## 3. 调用 Agent（SSE）

将 `<session-id>` 替换为上一步返回的会话 ID，将 `agent_id` 替换为目标 Agent ID：

```bash
curl -N -sS -X POST "$XELORA_URL/agent-chat/<session-id>" \
  -H "X-API-Key: $XELORA_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "query": "JP1/AJS3 中如何确认作业网络的执行状态？",
    "agent_enabled": true,
    "agent_id": "585d7108-b55b-4be5-86c1-ba53a200c86b",
    "web_search_enabled": false,
    "channel": "api"
  }'
```

BSP-RM 账票 Agent 只需替换 `agent_id`：

```text
dd87a237-9b65-4fe2-a7d4-5a4dbf995c8a
```

不要在请求中传入其他知识库 ID；Agent 已固定为单知识库检索。

## 4. 处理 SSE 响应

响应由多条 `data:` 事件组成，重点关注：

| `response_type` | 说明 |
|---|---|
| `references` | 检索到的知识库引用 |
| `tool_call` / `tool_result` | Agent 检索、深读等过程 |
| `answer` | 最终回答；`done=true` 表示结束 |
| `error` | 调用失败 |

事实性回答应包含内联引用，例如：

```text
作业网络的执行状态可通过相关 JP1 命令确认。<kb doc="文档名" chunk_id="..." />
```

客户端应持续读取 SSE，直到收到 `response_type=answer` 且 `done=true`，或收到 `response_type=error`。不要只读取第一条事件。

## 5. 清理会话

验证或一次性调用完成后，可删除会话：

```bash
curl -sS -X DELETE "$XELORA_URL/sessions/<session-id>" \
  -H "X-API-Key: $XELORA_API_KEY"
```

## 6. 常见错误

| HTTP / 返回 | 处理 |
|---|---|
| `401 Unauthorized` | 检查 Base URL、API Key 是否有效，以及请求头是否为 `X-API-Key`。 |
| `1003 Session not found` | 先调用 `POST /sessions`，使用返回的会话 ID。 |
| SSE 中没有 `references` | 知识库可能没有匹配证据；应以 Agent 的证据不足说明为准，不要自行补充外部知识。 |
| 网络超时 | 保持 SSE 长连接，适当增加客户端读取超时；不要重复提交同一会话请求。 |
