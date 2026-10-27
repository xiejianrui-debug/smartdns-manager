# SmartDNS Manager

维护此 fork 前请阅读 [项目维护指南](docs/maintenance.md)，其中记录代码位置、配置生效流程、提交合并要求和部署回退方式。开发代理入口为 [AGENTS.md](AGENTS.md)。

<div align="center">

[![Author](https://img.shields.io/badge/Author-yantao-blue.svg?style=flat-square)](https://github.com/almightyyantao)
[![Stars](https://img.shields.io/github/stars/almightyyantao/smartdns-manager?style=flat-square&logo=github)](https://github.com/almightyyantao/smartdns-manager/stargazers)
[![Forks](https://img.shields.io/github/forks/almightyyantao/smartdns-manager?style=flat-square&logo=github)](https://github.com/almightyyantao/smartdns-manager/network)
[![License](https://img.shields.io/github/license/almightyyantao/smartdns-manager?style=flat-square)](LICENSE)
[![Issues](https://img.shields.io/github/issues/almightyyantao/smartdns-manager?style=flat-square&logo=github)](https://github.com/almightyyantao/smartdns-manager/issues)
[![Last Commit](https://img.shields.io/github/last-commit/almightyyantao/smartdns-manager?style=flat-square&logo=github)](https://github.com/almightyyantao/smartdns-manager/commits/main)

**一个功能强大的 SmartDNS 集中管理平台**

支持多节点管理 · 配置同步 · 实时监控 · 消息推送

[快速开始](#-快速开始) · [功能特性](#-主要特性) · [系统截图](#-系统截图) · [文档](https://github.com/almightyyantao/smartdns-manager/wiki) · [反馈问题](https://github.com/almightyyantao/smartdns-manager/issues)

![Dashboard](docs/images/dashboard.png)

</div>

---

## ✨ 主要特性

### 🎯 核心功能

- **多节点管理** - 集中管理多个 SmartDNS 节点，统一配置和监控
- **配置同步** - 自动同步配置到所有节点，支持批量操作
- **实时监控** - 实时查看节点状态、资源使用情况、日志解析和服务运行状态
- **可视化配置** - 通过 Web 界面直观地管理 DNS 服务器和地址映射
- **配置备份** - 自动备份配置文件，支持一键恢复
- **消息推送** - 支持企业微信、钉钉、飞书等多种通知渠道

### 🔧 管理功能

-  DNS 服务器管理（UDP/TCP/TLS/HTTPS）
-  地址映射管理（域名到 IP 映射）
-  域名集管理（分组管理域名）
-  域名规则管理
-  配置模板管理
-  批量导入导出

### 🚀 运维功能

-  一键初始化节点（自动安装 SmartDNS）
-  远程重启服务
-  日志实时查看
-  配置同步状态追踪
-  节点健康检查
-  性能监控（CPU、内存、磁盘）

### 📱 通知功能

-  配置同步成功/失败通知
-  节点上线/离线通知
-  服务异常告警
-  支持企业微信、钉钉、飞书、Slack
-  自定义事件订阅

## 📸 系统截图

### 仪表板
![Dashboard](docs/images/dashboard.png)
### 节点管理
![Nodes](docs/images/nodes.png)
### 配置管理
![Config](docs/images/config.png)
### 通知管理
![Notification](docs/images/notification.png)

## 🏗️ 技术栈

<table>
<tr>
<td>

### 后端
- **Go 1.21+** - 高性能后端服务
- **Gin** - Web 框架
- **GORM** - ORM 框架
- **SQLite** - 轻量级数据库
- **SSH** - 远程节点管理

</td>
<td>

### 前端
- **React 18** - 用户界面
- **Ant Design 5** - UI 组件库
- **Axios** - HTTP 客户端
- **React Router** - 路由管理
- **Monaco Editor** - 代码编辑器

</td>
</tr>
</table>

## 🚀 快速开始

### 方式一：Docker Compose（推荐）

#### 前置要求
- Docker 20.10+
- Docker Compose 2.0+

#### 🚀 快速使用

```bash
# 拉取最新镜像
docker pull ghcr.io/almightyYantao/smartdns-manager:0.0.1

# 或使用 latest 标签
docker pull ghcr.io/almightyYantao/smartdns-manager:latest
```

#### 🏗️ 支持架构

- linux/amd64 (x86_64)
- linux/arm64 (ARM64/aarch64)

### 部署步骤

**1. 克隆项目**
```bash
git clone https://github.com/almightyyantao/smartdns-manager.git
cd smartdns-manager
```

### clickhouse 配置

如果要开启日志监听的话，最好是配置clickhouse，sqllite的日志监听性能会比较差

```bash
LOG_STORAGE_TYPE=clickhouse
CLICKHOUSE_HOST=xxx.xxx.xxx.xxx
CLICKHOUSE_PORT=9000
CLICKHOUSE_DB=smartdns_logs
CLICKHOUSE_USER=smartdns
CLICKHOUSE_PASSWORD=your_password
```

### docker compose 新增以下内容

```dockerfile
  clickhouse:
    image: clickhouse/clickhouse-server:latest
    container_name: smartdns-clickhouse
    ports:
      - "8123:8123"  # HTTP
      - "9000:9000"  # Native
    environment:
      CLICKHOUSE_DB: smartdns_logs
      CLICKHOUSE_USER: smartdns
      CLICKHOUSE_PASSWORD: your_password
    volumes:
      - ./clickhouse/data:/var/lib/clickhouse
      - ./clickhouse/logs:/var/log/clickhouse-server
    ulimits:
      nofile:
        soft: 262144
        hard: 262144
```

---

## 🤝 贡献

欢迎贡献代码！请查看 [贡献指南](CONTRIBUTING.md)。

## 📝 开源协议

本项目基于 [MIT License](LICENSE) 开源。

## ⭐ Star History

[![Star History Chart](https://api.star-history.com/svg?repos=almightyyantao/smartdns-manager&type=Date)](https://star-history.com/#almightyyantao/smartdns-manager&Date)

---

<div align="center">

**如果这个项目对你有帮助，请点个 ⭐ Star 支持一下！**

Made with ❤️ by [yantao](https://github.com/almightyyantao)

</div>
