# SmartDNS Admin 开发环境指南

本页部分环境要求和端口来自旧版。维护当前 fork 时，先读 [维护指南](docs/maintenance.md)，以 `go.mod`、`ui/package-lock.json` 及实际配置为准。

## 快速启动

我们提供了多种方式来启动开发环境，选择适合你的操作系统和偏好的方式：

### 🚀 一键启动脚本

#### Linux/macOS (推荐)
```bash
# 设置执行权限
chmod +x dev-start.sh

# 启动开发环境
./dev-start.sh
```

#### Windows CMD
```cmd
dev-start.bat
```

#### Windows PowerShell
```powershell
# 可能需要设置执行策略
Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser

# 启动开发环境
.\dev-start.ps1

# 跳过依赖安装（如果已安装）
.\dev-start.ps1 -SkipInstall
```

#### 使用 Makefile (推荐)
```bash
# 查看所有可用命令
make help

# 初始化开发环境
make setup

# 启动完整开发环境
make dev

# 只启动后端
make dev-backend

# 只启动前端
make dev-frontend
```

## 手动启动

如果你喜欢手动控制，可以分别启动：

### 后端服务
```bash
cd backend
go run main.go
```
- 访问地址: http://localhost:8080

### 前端服务
```bash
cd ui
npm install  # 首次运行需要安装依赖
npm start
```
- 访问地址: http://localhost:3000

## 环境要求

### 必需依赖
- **Go**: 1.19+ (用于后端开发)
- **Node.js**: 16+ (用于前端开发)
- **npm**: 8+ (Node.js 包管理器)

### 可选工具
- **Make**: 用于使用 Makefile 命令
- **Docker**: 用于容器化部署

## 脚本说明

### dev-start.sh (Linux/macOS)
- 自动检查依赖环境
- 自动安装前端依赖（如果不存在）
- 同时启动后端和前端服务
- 支持 Ctrl+C 优雅停止所有服务
- 彩色输出，易于查看状态

### dev-start.bat (Windows CMD)
- Windows 命令提示符版本
- 检查必要的依赖
- 分别在新窗口中启动后端和前端
- 手动关闭窗口停止服务

### dev-start.ps1 (Windows PowerShell)
- PowerShell 版本，功能最完整
- 支持参数选项（如 -SkipInstall）
- 使用 PowerShell Job 管理进程
- 支持优雅停止

### Makefile
- 提供简洁的命令接口
- 支持并行启动 (`make -j2`)
- 包含构建、测试、清理等完整工作流

## 使用示例

### 第一次使用
```bash
# 方式1: 使用 Makefile
make setup && make dev

# 方式2: 使用脚本
chmod +x dev-start.sh
./dev-start.sh

# 方式3: Windows
dev-start.bat
```

### 日常开发
```bash
# 最简单的方式
make dev

# 或者
./dev-start.sh
```

### 只启动某个服务
```bash
# 只启动后端
make dev-backend
# 或
cd backend && go run main.go

# 只启动前端  
make dev-frontend
# 或
cd ui && npm start
```

## 故障排除

### 权限问题 (Linux/macOS)
```bash
chmod +x dev-start.sh
```

### PowerShell 执行策略 (Windows)
```powershell
Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser
```

### 端口冲突
- 后端默认端口: 8080
- 前端默认端口: 3000
- 如果端口被占用，脚本会显示错误信息

### 依赖问题
```bash
# 清理并重新安装前端依赖
make clean && make setup

# 或手动清理
rm -rf ui/node_modules ui/package-lock.json
cd ui && npm install
```

## 项目结构

```
smartdns-admin/
├── backend/           # 后端 Go 代码
├── ui/               # 前端 React 代码
├── agent/            # 日志收集代理
├── dev-start.sh      # Linux/macOS 启动脚本
├── dev-start.bat     # Windows CMD 启动脚本  
├── dev-start.ps1     # Windows PowerShell 启动脚本
├── Makefile          # Make 命令配置
└── DEVELOPMENT.md    # 本开发指南
```

现在你可以选择最适合你环境的方式来启动开发环境了！
