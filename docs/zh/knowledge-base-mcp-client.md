# 知识库 MCP 端点：IDE / 客户端调用说明

本文描述 **Xelora「知识库 MCP 发布」** 对外暴露的 MCP 服务，供 Cursor、VS Code（Copilot MCP）、Claude Desktop、Windsurf 等任意支持 **HTTP Streamable MCP** 的客户端连接。

> **与其它 MCP 产品的区别**
>
> | 能力 | 知识库 MCP 端点（本文） | 内置 MCP 服务管理 | `mcp-server/` Stdio 服务 |
> |------|-------------------------|-------------------|---------------------------|
> | 用途 | 只读/可选写入**已发布知识库** | 连接**外部** MCP 服务器 | 本地进程桥接 REST API |
> | 连接 | `https://host/mcp/{endpoint_id}` + Bearer | 各服务自有 URL/命令 | `command` + `args` |
> | 配置入口 | 设置 → 知识库 MCP 发布 | 设置 → MCP 服务 | 见 `mcp-server/MCP_CONFIG.md` |

---

## 1. 使用前准备（Xelora 侧）

1. 使用**工作空间管理员**登录 Xelora Web。
2. 打开 **设置 → 数据与扩展 → 知识库 MCP 发布**。
3. **发布端点**：填写名称、绑定的知识库（或「全部知识库」）、勾选暴露的工具、可选默认 Agent（供 `ask`）、可选速率限制。
4. 保存后复制：
   - **MCP URL**（形如 `http(s)://<站点>/mcp/<endpoint_uuid>`）
   - **MCP 访问令牌**（仅显示一次，形如 `mcp_…`）
5. 若令牌泄露或人员变动，使用 **轮换令牌**；旧令牌立即失效。

**URL 说明**

- 路径固定为 `/mcp/{endpoint_id}`，`endpoint_id` 为端点 UUID。
- 界面展示的 URL 基于当前浏览器访问的 **Origin**（例如经反代时为 `http://localhost/mcp/…`，直连 API 可能为 `http://host:8080/mcp/…`）。客户端必须使用 **与 IDE 运行环境网络可达** 的同一 Origin。

---

## 2. 协议与认证（所有 IDE 通用）

### 2.1 传输方式

- **HTTP Streamable MCP**（JSON-RPC 2.0 over POST）。
- 服务端实现：`internal/mcpserver`（`xelora` / `1.0.0`）。
- **业务请求请使用 POST**；对同一 URL 的 **GET 可能建立 SSE 长连接**，不适合作为健康检查（易导致客户端一直等待或 curl 超时）。

### 2.2 必需 HTTP 头

| Header | 值 |
|--------|-----|
| `Authorization` | `Bearer <MCP 访问令牌>`（**仅 Header**；**不支持** Query 传 token） |
| `Content-Type` | `application/json` |
| `Accept` | `application/json, text/event-stream` |

### 2.3 鉴权错误

| HTTP | 含义 |
|------|------|
| 401 | 缺少或无效 Bearer |
| 403 | 端点已禁用或工作空间不可用 |

---

## 3. 标准调用流程（JSON-RPC）

客户端（IDE 内置 MCP 插件）通常自动完成下列步骤；手工调试时可按序发送。

```text
1. POST  initialize          → 获取 serverInfo、instructions
2. POST  tools/list          → 获取当前端点允许的工具列表（受端点勾选限制）
3. POST  tools/call          → 调用具体工具
   （可选）notifications/initialized 等 — 视客户端 MCP 库而定；多数场景 1→3 即可工作
```

### 3.1 `initialize` 示例

**请求**

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "initialize",
  "params": {
    "protocolVersion": "2024-11-05",
    "capabilities": {},
    "clientInfo": { "name": "my-ide", "version": "1.0" }
  }
}
```

**响应 `result` 字段（节选）**

- `serverInfo.name`: `xelora`
- `instructions`: 建议使用 `list_knowledge_bases`、`grep_chunks`、`search_knowledge`、`read_document`、`ask` 等

### 3.2 `tools/list`

**请求**

```json
{ "jsonrpc": "2.0", "id": 2, "method": "tools/list", "params": {} }
```

返回的 `result.tools[].name` 仅为**该端点已启用**的工具（未勾选的工具不会出现，调用也会被拒绝）。

### 3.3 `tools/call`

**请求结构**

```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "method": "tools/call",
  "params": {
    "name": "<工具名>",
    "arguments": { }
  }
}
```

**响应结构（常见）**

```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "result": {
    "content": [{ "type": "text", "text": "..." }],
    "structuredContent": { }
  }
}
```

- `content[0].text`：给模型阅读的文本（如 grep 的 XML、文档 chunk）。
- `structuredContent`：结构化字段（如命中数、chunk 列表），视工具而定。

---

## 4. 工具一览与推荐使用顺序

| 工具名 | 类型 | 说明 |
|--------|------|------|
| `list_knowledge_bases` | 检索 | **首选**：查看本端点可见知识库 ID/名称/能力 |
| `grep_chunks` | 检索 | 正则/关键词搜 chunk；**命令名、错误码、标识符**优先用它 |
| `search_knowledge` | 检索 | 语义检索；依赖向量/嵌入，可能较慢，超时需排查后端 |
| `list_documents` | 检索 | 列出某知识库内文档 |
| `read_document` | 检索 | 按 `knowledge_id` + chunk `offset`/`limit` 分页读正文 |
| `ask` | 对话 | 自然语言 `question`，由工作空间 Agent 合成答案（可带引用） |
| `wiki_search` / `wiki_read_page` / `wiki_index` | Wiki | 仅当知识库已生成 Wiki 时有意义 |
| `add_document` / `update_document` / `delete_document` | 写入 | **破坏性**；端点默认不勾选；启用后才会出现在 `tools/list` |

**推荐 Agent 策略（可写入 IDE 规则）**

1. `list_knowledge_bases` 确认 scope  
2. 精确词 → `grep_chunks`  
3. 概念问题 → `search_knowledge` 或 `ask`  
4. 需完整章节 → `read_document`（grep 结果中的 `knowledge_id`、chunk 索引辅助定位 offset）

**`grep_chunks` 参数示例**

```json
{
  "name": "grep_chunks",
  "arguments": {
    "query": "SNDMSG",
    "knowledge_base_ids": ["Manual_ASP"]
  }
}
```

`knowledge_base_ids` 可填 **UUID 或知识库精确名称**；省略则搜索端点 scope 内全部知识库。

**`ask` 参数示例（唯一常用「自然语言提示」字段）**

```json
{
  "name": "ask",
  "arguments": {
    "question": "概述 SNDMSG 命令的功能与主要参数",
    "knowledge_base_ids": ["Manual_ASP"]
  }
}
```

---

## 5. 各 IDE 配置要点

各产品配置文件路径不同，但字段语义一致：**Remote MCP URL + Bearer Header**。

### 5.1 Cursor

- **项目级**：`<workspace>/.cursor/mcp.json`
- **用户级**：以 Cursor 设置 → MCP 所示路径为准（版本间可能为 `~/.cursor/mcp.json` 等）

```json
{
  "mcpServers": {
    "xelora-kb": {
      "url": "http://localhost/mcp/a1ab5698-7fc1-4389-b56a-7ee13ca65592",
      "headers": {
        "Authorization": "Bearer mcp_xxxxxxxx"
      }
    }
  }
}
```

修改后：**Settings → MCP** 中刷新或重载窗口。聊天里 Agent 会自动 `tools/call`；用户自然语言**不会**原样发送到 Xelora（除非走 `ask`）。

### 5.2 VS Code / GitHub Copilot（MCP）

在支持的版本中，使用 MCP 配置文件（如 `.vscode/mcp.json` 或用户级 MCP 配置，以当前 VS Code 文档为准），同样填写 `url` + `headers.Authorization`。

### 5.3 Claude Desktop

较新版本支持 Remote MCP；配置 `mcpServers` 下服务器的 **URL 与 headers**（具体 JSON 键名以 Anthropic 当前文档为准）。旧版若仅支持 Stdio，则**不能**直接连知识库 HTTP 端点，需换用支持 Streamable HTTP 的客户端。

### 5.4 其它 IDE（Windsurf、Zed 等）

选择传输类型：**HTTP Streamable** / **Streamable HTTP**（**不要**选纯 SSE 端点 URL，除非产品明确兼容 Streamable）。  
填入发布页复制的 URL 与 Bearer。

---

## 6. 注意事项（务必写进内部规范）

### 6.1 安全

- 令牌等同**只读/部分写入** API 密钥：**勿提交 Git**、勿贴到公开文档；用环境变量或密钥管理。
- 不支持 URL `?token=`；避免令牌进访问日志。
- 端点可绑定**部分知识库**；客户端只能访问绑定范围内数据。
- 启用写入工具后，Agent 可能修改知识库内容；默认仅开放只读工具。

### 6.2 网络与 URL

- IDE 与 Xelora **必须互通**（本机 Docker、局域网 IP、HTTPS 反代均可）。
- `localhost` 在远程 SSH 开发容器中可能指向容器自身，应改为宿主机 IP 或 `host.docker.internal`。
- 前端 Origin（`:8081` / `:80`）与 API（`:8080`）若不同，以发布页显示的 **MCP URL** 为准。

### 6.3 性能与限流

- 端点可配置 **每分钟速率限制**；超限后 `tools/call` 会返回工具级错误（含 rate limit 提示）。
- `grep_chunks` 通常秒级；`search_knowledge` / `ask` 可能数十秒以上，客户端 HTTP 超时需适当放大。
- 大文档 `read_document` 请分页（`limit` 建议 ≤100），避免一次拉取过多 chunk。

### 6.4 行为语义

- **无端点上下文或未启用工具**时，`tools/list` 可能为空或调用返回 *not enabled*。
- **`read_document` 的 `offset`** 为 chunk 分页偏移，与 PDF 页码无简单 1:1 关系；需结合 `grep_chunks` 的 `chunk_index` 试探偏移。
- 聊天中的「提示词」在检索类工具里体现为 **`query` / `question` 等参数字段**，不是整段对话历史。

### 6.5 运维

- 端点 **禁用** 后现有连接应失败（403）。
- **轮换令牌** 后需同步更新所有 IDE 配置。
- 升级 Xelora 后若 MCP 协议版本变更，以 `initialize` 返回的 `protocolVersion` 为准。

---

## 7. 手工验证（curl）

Windows 请使用 `curl.exe`，JSON 建议放文件避免 PowerShell 转义问题。

```bash
curl.exe -X POST ^
  -H "Authorization: Bearer mcp_YOUR_TOKEN" ^
  -H "Content-Type: application/json" ^
  -H "Accept: application/json, text/event-stream" ^
  -d "@request.json" ^
  "http://localhost/mcp/YOUR_ENDPOINT_ID"
```

`request.json` 内容为上一节的 `initialize` / `tools/call` 等 JSON-RPC。

期望：**HTTP 200** + JSON body（或 SSE 包装的 JSON，客户端会自动解析 `data:` 行）。

---

## 8. 故障排查

| 现象 | 可能原因 | 处理 |
|------|----------|------|
| 401 unauthorized | 未带 Bearer 或令牌错误/已轮换 | 检查 headers；在 Xelora 轮换并更新配置 |
| 403 disabled | 端点或工作空间停用 | 管理台启用端点/空间 |
| 连接一直 pending | 用 GET 探测或 SSE 挂起 | 改用 POST + JSON-RPC |
| tools/list 为空 | 端点未勾选任何工具 | 编辑端点勾选工具 |
| search 超时 | 向量库/嵌入慢或不可用 | 先用 grep；查 Xelora 后端与 docreader 日志 |
| IDE 看不到服务器 | 配置路径错误或未重载 MCP | 确认 mcp.json；重载窗口 |
| 知识库内容不对 | scope 仅绑定部分 KB | `list_knowledge_bases` 核对 |

---

## 9. 参考

- 服务端路由与鉴权：`internal/router/routes_mcp_endpoint.go`、`internal/middleware/mcp_endpoint_auth.go`
- 工具定义：`internal/types/mcp_endpoint_tools.go`
- 管理 API（创建/轮换令牌，需登录 Xelora）：`/api/v1/mcp-endpoints`（Swagger / 管理界面）
- 本地调用样例日志（示例）：工作区 `logs/mcp-demo/MCP-CALL-TRACE.md`

---

## 10. 文档变更记录

| 日期 | 说明 |
|------|------|
| 2026-10-19 | 首版：HTTP Streamable、IDE 配置、工具与注意事项 |
