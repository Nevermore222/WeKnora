# WeKnora 二开开发指南

本文档是二开部署与开发的唯一参考,涵盖环境准备、源码构建、日常开发闭环和多机协作。
分支与 PR 规范见 [WORKFLOW.md](./WORKFLOW.md),改动追踪见 [CHANGELOG-custom.md](./CHANGELOG-custom.md)。

## 仓库结构

```
F:\Docker\WeKnora\WeKnora-main
├─ origin    git@github.com:Nevermore222/WeKnora.git   (你的 fork, SSH)
├─ upstream  https://github.com/Tencent/WeKnora.git     (只读, 拉上游更新)
├─ tag v0.6.2 @ fc98f00  (上游参考锚点)
├─ main              ← 上游同步层, 禁止直接改业务代码
└─ custom/business   ← 业务主分支, 所有二开改动汇聚于此
```

## 环境要求

- Docker Desktop (含 docker compose v2)
- Node.js >= 18 (当前 v24) + npm
- Git + SSH key 已配置 GitHub
- Go >= 1.26 (仅本地直接编译时需要,Docker 构建不需要)
- Python 3.10 (仅本地直接运行 docreader 时需要,Docker 构建不需要)

日常二开只需 Docker + Node + Git,Go/Python 编译都在 Docker 容器内完成。

## 首次部署(从零开始)

```powershell
# 1. 克隆仓库
git clone git@github.com:Nevermore222/WeKnora.git
cd WeKnora
git remote add upstream https://github.com/Tencent/WeKnora.git
git checkout custom/business

# 2. 配置环境变量
cp .env.example .env
# 编辑 .env, 填入 DB_PASSWORD / REDIS_PASSWORD / JWT_SECRET 等

# 3. 构建前端静态产物(前端 Dockerfile 依赖 dist, 必须先构建)
cd frontend
npm ci
npm run build
cd ..

# 4. 全量源码构建并启动
docker compose up -d --build

# 5. 等待健康检查通过
docker compose ps
```

访问验证:
- 前端: http://localhost
- 后端健康检查: http://localhost:8080/health

## 源码构建的服务对照表

| 服务 | 改哪部分源码 | 构建上下文 | Dockerfile | 重建命令 |
|---|---|---|---|---|
| app | Go 后端 (`cmd/` `internal/` `config/` 等) | 仓库根 | `docker/Dockerfile.app` | `docker compose up -d --build app` |
| frontend | 前端 (`frontend/src/` 等) | `./frontend` | `frontend/Dockerfile` | 先 `npm run build` 再 `docker compose up -d --build frontend` |
| docreader | 文档解析 Python (`docreader/`) | 仓库根 | `docker/Dockerfile.docreader` | `docker compose up -d --build docreader` |

postgres 和 redis 用官方镜像,不涉及源码构建。

## 日常开发闭环

### 1. 同步并派生分支

```powershell
cd F:\Docker\WeKnora\WeKnora-main
git checkout custom/business
git pull --rebase
git checkout -b feature/<模块>-<动作>
```

### 2. 改代码并提交

```powershell
git add -A
git commit -m "feat: <做了什么>"
git push -u origin feature/<模块>-<动作>
```

### 3. 重建部署验证

改了后端:
```powershell
docker compose up -d --build app
docker compose logs -f app
```

改了前端:
```powershell
cd frontend
npm run build
cd ..
docker compose up -d --build frontend
```

改了 docreader:
```powershell
docker compose up -d --build docreader
docker compose logs -f docreader
```

改了多个或不确定改了哪些:
```powershell
docker compose up -d --build
```

### 4. 验证通过后合并

在 GitHub 上开 PR: base=`custom/business`, compare=`feature/<模块>-<动作>`。
审查通过后合并, 然后同步本地:

```powershell
git checkout custom/business
git pull --rebase
git branch -d feature/<模块>-<动作>
```

在 [CHANGELOG-custom.md](./CHANGELOG-custom.md) 追加一条改动记录。

## 前端构建的关键点

前端 Dockerfile 是 `COPY dist /usr/share/nginx/html`, 不是在容器内跑 vite build。
所以每次改了前端代码, 必须先在本地构建 dist 再重建镜像:

```powershell
cd frontend
npm run build
cd ..
docker compose up -d --build frontend
```

`frontend/dist` 已被 gitignore, 不会进仓库, 每台机器各自本地构建。

## 构建缓存与强制重建

Docker 有分层缓存, 改动少时只重建变更层, 速度快。如果遇到"改了代码但没生效":

```powershell
# 单服务强制无缓存重建
docker compose build --no-cache app
docker compose up -d app

# 全量无缓存重建(慢但彻底)
docker compose build --no-cache
docker compose up -d
```

## 多机协作规则

GitHub 是唯一真相源, 本地只是工作副本。

1. 开始工作前 `git pull --rebase`, 结束前 `git push`。
2. 同一分支绝不在两台机器上同时有未推送改动。
3. `git push` 被拒(非 fast-forward)时用 `git pull --rebase`, 绝不 `--force`。
4. feature 分支一次只归一台机器所有, 开发期间不切换机器。
5. 切换机器前确认: 改动已 commit + push, 工作树 clean。

## 新机器初始化

```powershell
git clone git@github.com:Nevermore222/WeKnora.git
cd WeKnora
git remote add upstream https://github.com/Tencent/WeKnora.git
git checkout custom/business
cp .env.example .env
# 编辑 .env
cd frontend; npm ci; npm run build; cd ..
docker compose up -d --build
```

SSH key 需在新机器上单独配置 GitHub。

## 上游升级

WeKnora 发新版时(本机 GitHub 直连不通, 需配镜像或开代理):

```powershell
git checkout main
git fetch upstream --tags
git merge upstream/v<新版tag>
git push
git checkout custom/business
git merge main
# 解决冲突, 优先保留业务改动, 但理解上游改了什么
docker compose up -d --build
```

详细冲突策略参考 [CHANGELOG-custom.md](./CHANGELOG-custom.md) 中每个改动的"影响上游合并"字段。

## 常用命令速查

```powershell
# 查看容器状态
docker compose ps

# 查看日志
docker compose logs -f app
docker compose logs --tail 100 frontend

# 重启单个服务(不重建)
docker compose restart app

# 停止全部
docker compose down

# 停止并删除数据卷(慎用, 会丢数据库数据)
docker compose down -v

# 进入容器调试
docker compose exec app sh
docker compose exec docreader bash

# 查看镜像构建历史
docker image history wechatopenai/weknora-app:latest
```

## 注意事项

- `.env` 已被 gitignore, 不含敏感配置, 每台机器各自维护
- `frontend/dist` 已被 gitignore, 每台机器各自本地构建
- `frontend/node_modules` 已被 gitignore
- 首次全量构建约 5-10 分钟(Go 编译 + Python 依赖 + 前端打包), 后续增量构建快很多
- `.env` 中 `APK_MIRROR_ARG=mirrors.tencent.com` 已配腾讯镜像加速构建
- 改了 `config/config.yaml` 后需重启 app(该文件通过 volume 挂载, 不需重建镜像): `docker compose restart app`
- 改了 `skills/preloaded/` 下的 skill 后需重启 app(同样 volume 挂载): `docker compose restart app`
