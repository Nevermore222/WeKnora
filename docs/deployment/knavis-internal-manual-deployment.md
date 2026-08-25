# Knavis 内部版人工部署手册

本物料用于在 Windows Docker Desktop 主机 `{{TARGET_SERVER_IP}}` 上部署 Knavis 内部版。服务端的知识库检索、嵌入、智能体工具链、权限和工作区执行逻辑保持现有 Xelora 运行时不变；内部版只替换发布侧品牌和前端静态资源。

## 1. 交付包内容

解压后不要移动子目录，保持以下结构：

```text
<release>/
  DEPLOYMENT-GUIDE.zh-CN.md
  README-KNAVIS-INTERNAL.txt
  SHA256SUMS.txt
  manifest.json
  client/
    Knavis Personal.exe
    Knavis-Personal-Setup-*.exe        # 如果构建机安装了 NSIS
  server/
    .env                               # 目标部署专用密钥，禁止提交 GitHub
    docker-compose.yml
    deploy.ps1
    deploy.cmd                         # 执行策略受限时使用
    verify.ps1
    verify.cmd                         # 执行策略受限时使用
    stop.ps1
    stop.cmd                           # 执行策略受限时使用
    import-offline-images.ps1
    offline-images/*.tar
    config/
    skills/preloaded/
    data/workspaces/
```

包内不包含 n8n、CL、COBOL、开发机数据库、开发机工作区和其他本地运行数据。`server/.env` 是本次部署生成的私密配置，不要发到公共仓库或群聊。

## 2. 9.177 主机前置条件

在 9.177 上确认：

1. Windows 10/11 x64，Docker Desktop 已安装并切换到 Linux containers，WSL2 后端正常。
2. Docker Desktop 正在运行，并且 `docker info` 可以成功返回。
3. Docker Compose v2 可用，建议执行 `docker compose version` 确认。
4. 主机至少预留 50 GB 可用空间，推荐 80 GB 以上。离线镜像归档约占 12.5 GB，Docker 解包、数据库和工作区还会继续占用空间。
5. TCP `{{FRONTEND_PORT}}`、`{{APP_PORT}}` 和沙箱服务端口 `8090` 没有被其他程序占用。默认网页入口是 80，API 是 8080。
6. Windows 防火墙允许局域网客户端访问网页端口和 API 端口。若只通过网页访问，至少开放 `{{FRONTEND_PORT}}`；客户端默认还会访问 `{{SERVER_ORIGIN}}`。

在 PowerShell 中执行：

```powershell
docker info
docker compose version
Test-NetConnection {{TARGET_SERVER_IP}} -Port {{APP_PORT}}
Get-NetTCPConnection -State Listen -LocalPort {{FRONTEND_PORT}},{{APP_PORT}},8090 -ErrorAction SilentlyContinue
```

模型依赖不包含在本离线包中。若管理员使用本机 Ollama，请在 9.177 先安装并启动 Ollama，确认主机 `11434` 端口可用，并准备好需要的模型；包内 `.env` 默认使用 `http://host.docker.internal:11434`。若使用云端模型，请在服务端管理后台配置供应商和模型，并确认 9.177 的出网、代理和证书策略；不要把 API key 写入 Git 或交付包。

## 3. 复制和完整性校验

将整个 release 目录复制到 9.177，例如 `D:\Knavis\{{RELEASE_TAG}}`，不要只复制 `server` 或 `client` 子目录。Windows 如果禁止直接运行 `.ps1`，请优先执行包内的 `.cmd` 启动器；启动器只对当前子进程使用 `ExecutionPolicy Bypass`，不修改系统策略。

```powershell
Set-Location D:\Knavis\{{RELEASE_TAG}}\server
.\verify.cmd
```

如需校验整个交付包的全部文件和镜像归档：

```powershell
.\verify.cmd -VerifyChecksums
```

完整哈希校验会读取约 12.5 GB 镜像归档，需要等待一段时间。校验失败时不要继续部署，先重新复制对应文件。

## 4. 判断是否需要替换现有 Xelora

当前 Compose 保留现有 Xelora 的技术容器名、网络名和外部数据卷名，以保证运行时兼容。因此同一台 9.177 上不能同时运行这套内部版和原 Xelora 版。部署前先确认当前容器归属：

```powershell
docker ps -a --filter name=Xelora-
docker volume ls
```

如果主机是首次部署，直接进入下一节。如果已有正式 Xelora 服务：

1. 记录当前 Compose 目录、`.env`、镜像 tag、容器状态和数据卷名称。
2. 复制旧的部署目录作为回滚材料，另行备份数据库和业务文件。
3. 确认当前服务允许短暂停机后，再执行带 `-AllowReplaceExisting` 的部署命令。
4. 严禁执行 `docker compose down -v`、`docker volume prune` 或 `docker system prune --volumes`。这些操作可能删除知识库、文件和数据库数据。

本包只给服务镜像增加 `{{RELEASE_TAG}}` 版本 tag，并不会改写构建机上的 Xelora `:latest` 镜像。服务器导入归档也不会联网拉取或重新构建服务镜像。

## 5. 部署服务端

### 5.1 首次部署

```powershell
Set-Location D:\Knavis\{{RELEASE_TAG}}\server
.\verify.cmd
.\deploy.cmd
```

部署脚本会依次导入 `offline-images/*.tar`、校验 Compose、执行 `docker compose up -d --no-build`，最后检查 API 和网页入口。这个过程可能因镜像归档大小持续数分钟。

### 5.2 有意替换原 Xelora

仅在已经完成上一节备份并确认停机窗口后执行：

```powershell
Set-Location D:\Knavis\{{RELEASE_TAG}}\server
.\deploy.cmd -AllowReplaceExisting
```

`-AllowReplaceExisting` 只允许 Compose 管理这套固定名称的服务，不会删除卷，也不会停止其他名称的容器。不要为了绕过检查而直接手工删除容器。

## 6. 部署验收

```powershell
Set-Location D:\Knavis\{{RELEASE_TAG}}\server
docker compose --env-file .env -f docker-compose.yml ps
Invoke-WebRequest http://127.0.0.1:{{APP_PORT}}/health -UseBasicParsing
Invoke-WebRequest http://127.0.0.1:{{FRONTEND_PORT}}/ -UseBasicParsing
```

从局域网另一台电脑访问：

```text
网页管理端：{{FRONTEND_ORIGIN}}
客户端默认 API：{{SERVER_ORIGIN}}
```

浏览器出现 Knavis 登录页后，再使用客户端登录。若 API 健康检查返回 401，不要直接判定服务损坏；应先确认网页入口、容器状态和登录请求是否正常。查看日志：

```powershell
docker compose --env-file .env -f docker-compose.yml logs --tail=200 app
docker compose --env-file .env -f docker-compose.yml logs --tail=200 frontend
docker compose --env-file .env -f docker-compose.yml logs --tail=200 docreader
```

## 7. 发放客户端

将 `client\Knavis Personal.exe` 或 NSIS 安装包发给公司内部 Windows 客户端。客户端已经内置默认服务地址 `{{SERVER_ORIGIN}}`，正常安装后无需用户自行配置服务器地址。

客户端首次启动的检查顺序：

1. 确认客户端电脑能访问 `{{SERVER_ORIGIN}}`。
2. 打开客户端，看到 Knavis 登录界面。
3. 注册、登录、加入工作区和管理员权限以服务端当前策略为准。
4. 若客户端版本和服务端 API 不匹配，停止发放该客户端，改用同一 release 目录的服务端与客户端。

## 8. 日常启停和更新

正常停止服务且保留数据：

```powershell
Set-Location D:\Knavis\{{RELEASE_TAG}}\server
.\stop.cmd
```

重新启动当前包：

```powershell
docker compose --env-file .env -f docker-compose.yml up -d --no-build
```

更新到新包时，先完成新包的 `verify.cmd -VerifyChecksums`，再按第四节备份旧目录和数据，使用新包的 `deploy.cmd -AllowReplaceExisting`。不要把新包的 `.env` 与旧包混用，除非已经确认密钥和数据库迁移策略。

工作区宿主机目录是：

```text
D:\Knavis\{{RELEASE_TAG}}\server\data\workspaces
```

不要在服务运行时直接删除该目录内容。知识库文件和数据库还可能位于 Docker 外部卷中，备份时必须同时备份数据库卷和业务文件。

## 9. 回滚

1. 停止当前包：`.\stop.cmd`。
2. 切换回已备份的旧部署目录和旧 `.env`。
3. 使用旧包的镜像 tag 启动：`docker compose up -d --no-build`。
4. 检查旧包网页、API、登录、知识库和工作区数据。

回滚期间不要删除外部数据卷。当前包没有修改原 Xelora `:latest` 镜像；如果旧包引用的是原 tag，按旧包记录恢复即可。

## 10. 常见故障处理

| 现象 | 处理 |
| --- | --- |
| `Existing Xelora containers detected` | 确认是否要替换正式服务。要替换则先备份，再使用 `-AllowReplaceExisting`。 |
| 端口绑定失败 | 使用 `Get-NetTCPConnection` 找出占用程序；或在部署前修改 `.env` 的 `APP_PORT`/`FRONTEND_PORT`，并同步调整客户端默认地址。 |
| `No offline image archives found` | 确认复制的是完整 release 目录，并且 `server\offline-images` 下存在全部 `.tar`。 |
| 客户端提示 API contract 不兼容 | 客户端和服务端必须来自同一个 release；不要把当前客户端连接到未匹配的旧服务端。 |
| 网页打不开但容器运行 | 先检查 Windows 防火墙和网页端口，再看 `frontend` 日志；从 9.177 本机和局域网分别请求网页。 |
| 沙箱或工作区执行失败 | 检查 `XELORA_SANDBOX_MODE=docker`、`XELORA_WORKSPACE_HOST_ROOT`、`Xelora-app` 容器状态以及 Docker Desktop 的 Linux 容器模式。 |

## 11. 最终验收清单

- [ ] `verify.cmd` 通过，必要时 `-VerifyChecksums` 通过。
- [ ] `docker compose ps` 中 app、frontend、docreader、postgres、redis、opensandbox 服务正常。
- [ ] `http://{{TARGET_SERVER_IP}}:{{APP_PORT}}/health` 和 `{{FRONTEND_ORIGIN}}` 可访问。
- [ ] 客户端默认连接 `{{SERVER_ORIGIN}}`，登录页显示 Knavis。
- [ ] 新用户权限由管理员设置生效，知识库检索链路未被包装过程改写。
- [ ] 已记录 `.env`、数据库、业务文件和工作区的备份位置。
- [ ] 未将 n8n、CL、COBOL 或 `server/.env` 提交到 GitHub。
