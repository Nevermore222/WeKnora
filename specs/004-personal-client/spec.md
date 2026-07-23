# 004 — Xelora Personal Client

Status: Draft
Created: 2026-07-21
Branch: 002-session-workspace-binding

## 1. Overview

Xelora Personal Client 是面向个人用户的桌面端 EXE 应用，在现有 "Xelora Lite"
Wails 桌面壳基础上演进。核心定位：

- **本地全功能**：内嵌 Go 后端 + SQLite + sqlite-vec，离线即可创建知识库、智能体、
  执行 Skill，无需依赖任何远程服务。
- **企业资源实时嵌入**：通过局域网连接管理员部署的 Xelora Server，实时调用企业级
  知识库检索、企业智能体对话、企业 Skill 执行。
- **双模 Skill 执行**：默认本地进程直接执行（Python/Node/Bash），用户可选开启
  Docker 沙箱隔离。

## 2. Goals

| # | Goal | Priority |
|---|------|----------|
| G1 | 个人用户零配置即可本地创建知识库、智能体、Skill 并执行 | P0 |
| G2 | 连接局域网企业服务器后，企业知识库/智能体/Skill 实时可用 | P0 |
| G3 | 本地资源与企业资源在 UI 中清晰区分、统一交互 | P0 |
| G4 | Skill 执行支持本地进程和 Docker 沙箱两种模式 | P1 |
| G5 | 单文件 EXE 分发，体积 < 50MB（不含前端资源） | P1 |
| G6 | 企业服务器断连时本地功能不受影响，企业资源优雅降级 | P1 |
| G7 | 支持自动更新（复用现有 update.go 机制） | P2 |

## 3. Non-Goals (v1)

- 多用户/多租户本地管理（个人客户端为单用户模式）
- 企业知识库向量索引离线分片下载
- 本地模型推理（LLM 推理仍走远程 API 或企业服务器代理）
- 移动端适配
- 企业 Skill 的本地缓存执行（v1 企业 Skill 始终远程调用）

## 4. Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                    Xelora Personal Client (EXE)                  │
│                                                                  │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │              Wails v2 Shell (WebView2)                     │   │
│  │  ┌────────────────────────────────────────────────────┐   │   │
│  │  │         Vue 3.5 Frontend (existing)                 │   │   │
│  │  │  + Personal/Enterprise resource switcher            │   │   │
│  │  │  + Server connection manager UI                     │   │   │
│  │  └────────────────────────────────────────────────────┘   │   │
│  └──────────────────────────────────────────────────────────┘   │
│                          │ Reverse Proxy                         │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │              Embedded Go Backend (Gin)                     │   │
│  │                                                            │   │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────────┐  │   │
│  │  │ Local Core   │  │ Enterprise  │  │ Skill Executor   │  │   │
│  │  │              │  │ Connector   │  │                   │  │   │
│  │  │ • SQLite DB  │  │             │  │ • Local Process   │  │   │
│  │  │ • sqlite-vec │  │ • Server    │  │ • Docker Sandbox  │  │   │
│  │  │ • Local KB   │  │   Registry  │  │ • Provider Select │  │   │
│  │  │ • Local Agent│  │ • API Proxy │  │                   │  │   │
│  │  │ • Local Skill│  │ • Auth/Token│  │                   │  │   │
│  │  │ • Workspace  │  │ • Health    │  │                   │  │   │
│  │  └─────────────┘  └─────────────┘  └─────────────────┘  │   │
│  └──────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
                          │
                          │ HTTP (LAN)
                          ▼
┌─────────────────────────────────────────────────────────────────┐
│              Xelora Enterprise Server (existing)                  │
│  • Enterprise Knowledge Bases (PostgreSQL + Vector Store)        │
│  • Enterprise Agents                                             │
│  • Enterprise Skills (sandbox execution)                         │
│  • User/Auth management                                          │
└─────────────────────────────────────────────────────────────────┘
```

## 5. Module Design

### 5.1 Local Core (existing, adapt)

现有 `internal/` 全部模块在 `DB_DRIVER=sqlite` 下已可运行。个人客户端需要：

- 单用户模式：跳过租户/组织/邀请流程，启动时自动创建默认 tenant + user
- 本地知识库：复用现有 knowledgebase 模块，向量存储使用 sqlite-vec
- 本地智能体：复用现有 custom_agent 模块
- 本地 Skill：复用现有 `internal/agent/skills` + `skills/preloaded/`
- 本地工作区：复用现有 `internal/workspace` 模块

### 5.2 Enterprise Connector (new)

新增 `internal/enterprise/` 包，负责与远程企业服务器通信：

```go
package enterprise

// ServerConfig 存储企业服务器连接配置
type ServerConfig struct {
    ID          string `json:"id"`
    Name        string `json:"name"`
    BaseURL     string `json:"base_url"`      // e.g. http://192.168.1.100:8080
    APIToken    string `json:"api_token"`     // 认证 token
    AutoConnect bool   `json:"auto_connect"`  // 启动时自动连接
    LastSeenAt  *time.Time `json:"last_seen_at,omitempty"`
}

// Connector 管理与企业服务器的连接生命周期
type Connector struct {
    servers  map[string]*ServerConnection
    healthCh chan HealthEvent
}

// ServerConnection 表示一个活跃的企业服务器连接
type ServerConnection struct {
    Config     ServerConfig
    Status     ConnectionStatus  // connected, disconnected, error
    Client     *http.Client
    Capabilities *ServerCapabilities
}

// ServerCapabilities 描述企业服务器暴露的能力
type ServerCapabilities struct {
    KnowledgeBases []RemoteKnowledgeBase `json:"knowledge_bases"`
    Agents         []RemoteAgent         `json:"agents"`
    Skills         []RemoteSkill         `json:"skills"`
    Models         []RemoteModel         `json:"models"`
}
```

核心职责：
- **服务发现**：手动配置或 mDNS 自动发现局域网内的 Xelora Server
- **认证**：使用 API Token 或 OAuth 连接企业服务器
- **能力拉取**：连接后拉取企业知识库/智能体/Skill 列表
- **请求代理**：将企业资源的对话/检索请求代理到远程服务器
- **健康检查**：定期心跳，断连时标记企业资源为不可用
- **优雅降级**：服务器不可达时，企业资源显示为离线，本地功能不受影响

### 5.3 Skill Executor (adapt)

复用现有 `internal/executor/` 网关，增加执行模式选择：

```go
// ExecutionMode 控制 Skill 的执行环境
type ExecutionMode string

const (
    ExecutionModeLocal  ExecutionMode = "local"   // 本地进程直接执行
    ExecutionModeDocker ExecutionMode = "docker"  // Docker 沙箱隔离执行
)
```

- 默认 `local`：Skill 脚本直接在用户机器上执行（Python/Node/Bash）
- 可选 `docker`：复用现有 ControlledDockerProvider，需要用户安装 Docker Desktop
- 企业 Skill：始终通过 Enterprise Connector 远程调用，不在本地执行

### 5.4 Frontend Adaptations

在现有 Vue 前端基础上增加：

- **资源来源标识**：知识库/智能体/Skill 列表项显示来源徽章（本地 / 企业）
- **服务器连接管理**：设置页新增"企业服务器"面板，支持添加/编辑/删除/测试连接
- **连接状态指示器**：顶栏显示企业服务器连接状态（已连接/断开/重连中）
- **企业资源只读标记**：企业知识库/智能体不可本地编辑，只能使用
- **Skill 执行模式切换**：Skill 执行时可选择本地/沙箱模式

## 6. Data Model Changes

### 6.1 本地新增表（SQLite）

```sql
-- 企业服务器配置（持久化到本地 SQLite）
CREATE TABLE enterprise_servers (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    base_url    TEXT NOT NULL,
    api_token   TEXT,
    auto_connect BOOLEAN DEFAULT 1,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- 企业资源缓存（加速 UI 渲染，非权威数据）
CREATE TABLE enterprise_resource_cache (
    id          TEXT PRIMARY KEY,
    server_id   TEXT NOT NULL REFERENCES enterprise_servers(id),
    resource_type TEXT NOT NULL,  -- 'knowledge_base', 'agent', 'skill', 'model'
    resource_id TEXT NOT NULL,
    metadata    TEXT,             -- JSON
    cached_at   DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

### 6.2 现有表适配

- `sessions` 表：增加 `source` 字段区分本地会话和企业代理会话
- `custom_agents` 表：增加 `origin` 字段（'local' | 'enterprise'）
- 知识库表：增加 `origin` 字段

## 7. API Design

### 7.1 本地 API（复用现有 /api/v1）

所有现有 API 保持不变，个人客户端内嵌后端直接服务。

### 7.2 新增企业连接 API

```
POST   /api/v1/enterprise/servers          # 添加企业服务器
GET    /api/v1/enterprise/servers          # 列出已配置服务器
PUT    /api/v1/enterprise/servers/:id      # 更新服务器配置
DELETE /api/v1/enterprise/servers/:id      # 删除服务器
POST   /api/v1/enterprise/servers/:id/test # 测试连接
POST   /api/v1/enterprise/servers/:id/connect    # 手动连接
POST   /api/v1/enterprise/servers/:id/disconnect # 手动断开
GET    /api/v1/enterprise/servers/:id/status     # 连接状态

GET    /api/v1/enterprise/resources        # 聚合所有已连接服务器的资源列表
GET    /api/v1/enterprise/resources/:type  # 按类型筛选 (knowledge_bases, agents, skills)
```

### 7.3 企业资源代理 API

```
POST   /api/v1/enterprise/chat             # 代理企业智能体对话
POST   /api/v1/enterprise/retrieval        # 代理企业知识库检索
POST   /api/v1/enterprise/skill/execute    # 代理企业 Skill 执行
```

## 8. Security

- API Token 存储在本地 SQLite，使用 OS keychain（Windows Credential Manager）加密
- 企业通信走 HTTP（局域网），可选 HTTPS（用户自签证书）
- 本地 Skill 执行默认在用户权限下运行，Docker 模式提供额外隔离
- 企业服务器不可信任本地客户端的任意请求，认证由服务器端控制

## 9. Build & Distribution

- 构建工具：Wails v2 CLI (`wails build -platform windows/amd64`)
- 输出：单文件 `Xelora-Personal.exe`（内嵌前端资源）
- 前端构建：`npm run build` → `frontend/dist/` → Wails embed
- 体积目标：< 50MB（Go binary ~30MB + frontend assets ~15MB）
- 自动更新：复用现有 `update.go`，指向个人客户端 release feed

## 10. Dependencies

| Dependency | Purpose | Status |
|-----------|---------|--------|
| Wails v2 | Desktop shell | Already in go.mod |
| SQLite + sqlite-vec | Local DB + vector search | Already in go.mod |
| WebView2 | Windows webview runtime | OS built-in (Win10+) |
| Docker Desktop | Optional sandbox execution | User-installed |
| Xelora Server | Enterprise resources | LAN deployed |

## 11. Migration Path

现有 "Xelora Lite" (`cmd/desktop`) → "Xelora Personal" 的演进路径：

1. 重命名产品标识（Xelora Lite → Xelora Personal）
2. 增加单用户自动初始化逻辑
3. 新增 `internal/enterprise/` 包
4. 前端增加企业连接管理和资源来源标识
5. Skill 执行增加模式选择
6. 打包为 Windows EXE

不破坏现有服务器端部署模式，个人客户端是独立的产品线。
