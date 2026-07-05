# Xelora 二开工作流

本文件定义多机轮流开发时的分支、同步、PR 规范。
所有机器在开始工作前应先 `git pull`,结束工作前先 `git push`。

## 分支模型

| 分支 | 用途 | 规则 |
|---|---|---|
| `main` | 上游同步层 | 只跟 Tencent/Xelora 同步,禁止直接改业务代码 |
| `custom/business` | 业务主分支 | 所有二开改动汇聚于此,多机共享,推前必拉 |
| `feature/<模块>-<动作>` | 短期需求分支 | 一次只在一台机器上开发,改完即删 |
| `hotfix/<问题>` | 紧急修复分支 | 从 custom/business 派生,改完即合回即删 |

命名示例:`feature/docreader-mylogic`、`hotfix/postgres-conn-leak`。

## 多机协作铁律

1. 同一分支绝不在两台机器上同时有未推送改动。
2. 每次开始工作前 `git pull --rebase`;每次结束前 `git push`。
3. `git push` 被拒(非 fast-forward)时用 `git pull --rebase`,绝不 `--force`。
4. feature 分支一次只归一台机器所有,开发期间不切换机器。

## PR 类型

| PR 方向 | 用途 |
|---|---|
| `feature/* -> custom/business` | 业务功能/二开改动 |
| `hotfix/* -> custom/business` | 紧急修复 |
| `main -> custom/business` | 上游升级同步 |

PR 标题用 Conventional Commits:`feat:` / `fix:` / `refactor:` / `docs:` / `chore:`。

## 标准开发流程

1. 同步并派生:
   ```powershell
   git checkout custom/business
   git pull --rebase
   git checkout -b feature/<模块>-<动作>
   ```
2. 改代码,提交:
   ```powershell
   git add -A
   git commit -m "feat: <做了什么>"
   ```
3. 推送并开 PR:
   ```powershell
   git push -u origin feature/<模块>-<动作>
   ```
   在 GitHub 上开 PR:base=`custom/business`,compare=`feature/<模块>-<动作>`。
4. 审查通过后合并 PR,删分支:
   ```powershell
   git checkout custom/business
   git pull --rebase
   git branch -d feature/<模块>-<动作>
   ```
5. 构建部署:
   ```powershell
   docker compose -f F:\Docker\Xelora\Xelora-main\docker-compose.yml up -d --build
   ```

## 上游升级流程

Xelora 发新版时(注意本机 GitHub 直连不通,需配镜像或开代理):

1. 同步上游到 main:
   ```powershell
   git checkout main
   git fetch upstream --tags
   git merge upstream/v<新版tag>
   git push
   ```
2. 开 PR `main -> custom/business`,在 GitHub 上审查上游改动差异。
3. 合并后在 custom/business 上解决冲突,重新构建验证。
4. 同步 CHANGELOG-custom.md。

## 切换机器前的检查清单

- [ ] 当前改动已 commit
- [ ] 已 push 到 origin
- [ ] 工作树 clean(`git status` 无未提交)
- [ ] 若在 feature 分支,已在 GitHub 开 PR 或记录分支名
