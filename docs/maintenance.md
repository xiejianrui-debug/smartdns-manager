# SmartDNS 控制台维护指南

本文件记录此 fork 的维护约定和实现边界，供后续开发、排障、提交与部署时读取。更新日期：2026-10-27。生产地址、认证入口和部署快照单独记录在本机的 `docs/maintenance.local.md`，不要提交该文件。

## 仓库与提交要求

| 用途 | 仓库或分支 |
| --- | --- |
| 原作者仓库，远程名 `origin` | `almightyYantao/smartdns-manager` |
| 用户维护的 fork，远程名 `fork` | `xiejianrui-debug/smartdns-manager` |
| fork 默认分支 | `main` |
| 新功能分支 | `codex/<用途>` |

先执行 `git status --short`、`git remote -v`、`git fetch fork`，确认现场。不得覆盖用户已有改动。以用户 fork 的最新 `main` 为基础开发；不要因本地 `main` 原先跟踪 `origin/main` 而误推上游。

1. 修改代码，完成与变更相称的测试和构建；界面变更要检查实际页面。
2. 提交用途清楚的 commit，并推送功能分支到 `fork`。本地 commit、推送分支、合并默认分支、部署生产要分别报告。
3. 用户确认合并或要求全部合并时，检查分支差异后合入 fork 的 `main`，推送并用 `git ls-remote fork refs/heads/main` 验证。
4. 默认保留分支；合并不等于删除。不同提交号可能包含相同补丁，应先用 `git cherry` 核对。旧草稿被新版完整替代时，要说明保留哪版实现，不能为消除分支差异回退修复。
5. 交付时给出 fork 的分支或 commit 链接、验证结果、是否合并以及是否部署。不要仅给出本地 SHA。

常用命令（将功能分支名替换为实际名称）：

```sh
git switch -c codex/<用途> fork/main
git add <本次修改的文件>
git commit -m '<说明本次修改用途>'
git push -u fork codex/<用途>
# 用户要求合并后，在确认工作区与分支关系后执行：
git switch main
git merge --ff-only fork/main
git merge --ff-only codex/<用途>
git push fork main
git ls-remote fork refs/heads/main
```

不能快进时先检查双方差异，按实际情况合并并处理冲突；不要 force push 或使用硬重置覆盖历史。

## 改代码从哪里入手

| 功能 | 主要位置 |
| --- | --- |
| 节点配置编辑、保存、重启按钮 | `ui/src/pages/NodeConfig.js` |
| 节点专属 DNS 视图与配置解析 | `ui/src/components/Config/NodeServerManager.js`、`nodeServers.js` |
| 全局 DNS 管理表格 | `ui/src/components/Config/ServerManager.js` |
| 地址映射及生效提示 | `ui/src/components/Config/AddressManager.js` |
| 日志上游、分组和静态映射标记 | `ui/src/components/Log/`、`agent/utils/static_address.go` |
| 配置 API 与 DNS 记录 API | `backend/handlers/config.go`、`server.go` |
| 配置解析、生成、完整同步 | `backend/services/parser.go`、`config_sync.go` |
| 本机定时任务 | `backend/services/custom_script_local.go`，另见 [本机任务说明](local-scheduled-scripts.md) |
| 运行容器 | `Dockerfile`、`nginx.conf`、`supervisord.conf`；实际 Compose 以部署现场为准 |

前端使用 React 18、Ant Design 5；实际依赖以 `ui/package-lock.json` 为准。Go 版本以各模块 `go.mod` 为准，当前要求 Go 1.24，backend 指定 toolchain 1.24.3。管理服务默认端口见 `backend/config/config.go`，当前为 3001；前端通常通过同源 `/api` 代理访问，不要把旧开发指南中的端口直接当成生产配置。

## DNS 配置怎样生效

顺序为：**管理台记录 → 节点配置文件 → SmartDNS 进程加载**。三者不能互相代替。

| 操作 | 当前行为 |
| --- | --- |
| 全局 DNS 添加、编辑、删除 | 更新数据库，不自动下发或重启节点 |
| 完整同步 | 读取已启用数据库记录，按 `node_ids` 过滤，再合并到节点文件；不自动重启 |
| 编辑器保存 | 备份并保存编辑器内容，不动态读取整张 DNS 表，不重启 |
| 重启服务 | 对选定节点执行 `systemctl restart smartdns`，加载磁盘配置，不保存未提交编辑内容 |
| 编辑器重新加载 | 从节点重读文件，不重启服务 |
| 节点 DNS 标签页 | 读取主配置文件中的上游；已分配但参数不同的数据库记录显示为“待同步” |

`node_ids` 为空或 `[]` 表示全局适用；指定节点列表时仅对列表内节点适用。主配置引用其他配置文件时，节点列表会提示自身只展示主文件直接声明的上游，不能当成全部运行状态。

完整同步当前按地址合并：更新同地址、追加新地址，不保证删除文件中已被数据库移除的旧条目。涉及删除或协议切换时，应核对实际文件，不能只修改数据库后重启。

已知边界：后端解析器目前主要识别 `server ...`；`server-https` 等显式协议指令必须一并处理，不能在 UI 解析时漏掉，也不能假设旧后端再生成配置会完整保留这些指令。DoH 的固定 IP、证书校验等高级选项还不在全局表单中，普通编辑可能重建并丢失这些参数；操作后必须核对数据库记录与文件。

“已写入配置”不等于“已加载”。核验重启应比较 PID、启动时间、systemd 日志，并检查上游连接和 DNS 查询，不能只信成功提示或缓存命中的查询。

## 验证与构建

```sh
cd ui
# 有锁文件且依赖未安装时使用 npm ci，不主动删除锁文件。
CI=true npm test -- --watchAll=false --runInBand
npm run build
```

只改节点列表时可先运行其专项测试：

```sh
cd ui
CI=true npm test -- --watchAll=false --runInBand --runTestsByPath src/components/Config/__tests__/nodeServers.test.js
```

```sh
cd backend
go test ./services -run 'Test(Local|CustomScript)' -count=1
```

Agent 变更时在 `agent` 中运行 `go test ./config ./utils`，按触及范围补充测试。构建有既有 warning 时说明是否新增；不要为小改动顺便升级依赖。前端检查至少覆盖目标节点正常读取、待同步、读取失败以及全局入口；预览使用只读接口或隔离数据。

## 部署与回退

生产部署须有用户明确指令。已获授权后完成以下步骤，不用再次询问同一次操作：

1. 检查运行容器的镜像 ID、Compose 实际文件、挂载卷、后端摘要与当前页面。运行镜像标签、Compose 标签、容器内单独更新的前端可能不一致。
2. 保留旧静态目录、Compose 文件及可恢复镜像。现有后端包含定制功能，不能直接换官方 `latest`。
3. 前端变更使用通过构建的产物，先发布哈希静态资源，最后原子替换 `index.html`；保留旧哈希资源，避免已打开页面请求失败。后端无需随静态文件发布而重启。
4. 把前端变更制作成可重建的持久化镜像，并更新实际 Compose 引用。只向容器复制文件不是完整发布；仅制作镜像也不代表运行页面已更新。
5. 验证线上首页与静态资源摘要、登录/API、目标节点实际页面。前端发布核对后端未变化，且未更改或重启 DNS 节点。
6. 在本机维护记录写下时间、源码 SHA、镜像标签、运行状态及回退位置；失败时恢复旧页面和 Compose。不要把凭据、原始日志或生产配置写进提交。

## 维护记录

- 2026-10-27：节点 DNS 页面改为读取当前节点主配置文件，增加待同步记录与全局管理入口；`da0d064` 已部署。
- 2026-10-27：用户要求归并已有分支，并建立本维护入口。日志分支的重复补丁保留历史，补入地址映射重启提示；本机任务旧草稿保留新版进程组取消和超时测试。
